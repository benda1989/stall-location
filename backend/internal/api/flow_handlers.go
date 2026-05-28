package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gkk/stall-location/backend/internal/auth"
	"github.com/gkk/stall-location/backend/internal/models"
	"github.com/gkk/stall-location/backend/internal/services"
	"gorm.io/gorm"
)

type sendSMSCodeRequest struct {
	Phone string `json:"phone" binding:"required"`
	Scene string `json:"scene" binding:"required"`
}

func (s *Server) SendSMSCode(c *gin.Context) {
	var req sendSMSCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	req.Scene = normalizeSMSScene(req.Scene)
	if req.Scene == "" {
		abort(c, http.StatusBadRequest, errors.New("scene must be merchant, admin or customer"))
		return
	}
	code := services.RandomDigits(6)
	if err := services.SendVerificationCode(c.Request.Context(), services.SMSConfig{
		Provider:        s.Config.SMS.Provider,
		AccessKeyID:     s.Config.SMS.AccessKeyID,
		AccessKeySecret: s.Config.SMS.AccessKeySecret,
		TemplateCode:    s.Config.SMS.TemplateCode,
		SignName:        s.Config.SMS.SignName,
		RegionID:        s.Config.SMS.RegionID,
	}, req.Phone, code); err != nil {
		abort(c, http.StatusBadGateway, err)
		return
	}
	record := models.SMSCode{
		Phone:     req.Phone,
		Scene:     req.Scene,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := s.DB.Create(&record).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	resp := gin.H{"sent": true, "scene": req.Scene, "expires_in_seconds": 300}
	if s.Config.GinMode != "release" {
		resp["dev_code"] = code
		resp["dev_hint"] = "本地开发环境也可使用 123456"
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) WeChatSilentStart(c *gin.Context) {
	redirect := strings.TrimSpace(c.Query("redirect"))
	if redirect == "" {
		redirect = s.Config.BaseURL + "/nearby"
	}
	if s.shouldUseDevWeChatLogin(c) {
		s.redirectWithCustomerToken(c, redirect, "dev-openid-"+devOpenIDSeed(c))
		return
	}
	if s.Config.WeChatAppID == "" {
		c.JSON(http.StatusOK, gin.H{
			"enabled":  false,
			"reason":   "WECHAT_APP_ID is not configured",
			"redirect": redirect,
		})
		return
	}
	callback := buildRequestOrigin(c) + "/api/wechat/oauth/callback?redirect=" + url.QueryEscape(redirect)
	oauthURL := "https://open.weixin.qq.com/connect/oauth2/authorize?appid=" + url.QueryEscape(s.Config.WeChatAppID) +
		"&redirect_uri=" + url.QueryEscape(callback) +
		"&response_type=code&scope=snsapi_base&state=stall-location#wechat_redirect"
	c.Redirect(http.StatusFound, oauthURL)
}

func (s *Server) WeChatOAuthCallback(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	redirect := strings.TrimSpace(c.Query("redirect"))
	if code == "" {
		abort(c, http.StatusBadRequest, errors.New("code is required"))
		return
	}
	if redirect == "" {
		redirect = s.Config.BaseURL + "/nearby"
	}

	openID, err := s.openIDFromWeChatCode(c.Request.Context(), code)
	if err != nil {
		abort(c, http.StatusBadGateway, err)
		return
	}
	token, user, err := s.customerTokenForOpenID(openID)
	if err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	if c.Query("json") == "1" {
		c.JSON(http.StatusOK, gin.H{"token": token, "role": models.RoleCustomer, "user": user, "openid": openID})
		return
	}
	s.redirectToCustomerTarget(c, redirect, token)
}

func (s *Server) redirectWithCustomerToken(c *gin.Context, redirect string, openID string) {
	token, _, err := s.customerTokenForOpenID(openID)
	if err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	s.redirectToCustomerTarget(c, redirect, token)
}

func (s *Server) redirectToCustomerTarget(c *gin.Context, redirect string, token string) {
	target, err := url.Parse(redirect)
	if err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	query := target.Query()
	query.Set("customer_token", token)
	target.RawQuery = query.Encode()
	c.Redirect(http.StatusFound, target.String())
}

func (s *Server) customerTokenForOpenID(openID string) (string, models.User, error) {
	var user models.User
	err := s.DB.Where("role = ? AND open_id = ?", models.RoleCustomer, openID).First(&user).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", user, err
		}
		user = models.User{Role: models.RoleCustomer, OpenID: openID, Nickname: "微信顾客"}
		if err := s.DB.Create(&user).Error; err != nil {
			return "", user, err
		}
	}
	token, err := auth.Sign(s.Config.TokenSecret, auth.Claims{UserID: user.ID, Role: models.RoleCustomer})
	if err != nil {
		return "", user, err
	}
	return token, user, nil
}

type weChatOAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

func (s *Server) openIDFromWeChatCode(ctx context.Context, code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", errors.New("code is required")
	}
	if s.Config.WeChatAppID == "" || s.Config.WeChatAppSecret == "" {
		// 本地未配置公众号密钥时，用 code 派生 dev openid，保证流程可闭环。
		return "dev-openid-" + code, nil
	}
	endpoint, _ := url.Parse("https://api.weixin.qq.com/sns/oauth2/access_token")
	query := endpoint.Query()
	query.Set("appid", s.Config.WeChatAppID)
	query.Set("secret", s.Config.WeChatAppSecret)
	query.Set("code", code)
	query.Set("grant_type", "authorization_code")
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", err
	}
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("wechat oauth exchange failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("wechat oauth exchange returned %d", resp.StatusCode)
	}
	var payload weChatOAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("wechat oauth response invalid")
	}
	if payload.ErrCode != 0 {
		return "", fmt.Errorf("wechat oauth exchange failed: %s", payload.ErrMsg)
	}
	if payload.OpenID == "" {
		return "", errors.New("wechat oauth response missing openid")
	}
	return payload.OpenID, nil
}

func (s *Server) shouldUseDevWeChatLogin(c *gin.Context) bool {
	if strings.EqualFold(s.Config.GinMode, "release") {
		return false
	}
	host := requestHost(c)
	if isLocalOAuthHost(host) {
		return true
	}
	redirect := strings.TrimSpace(c.Query("redirect"))
	if redirect == "" {
		return false
	}
	target, err := url.Parse(redirect)
	if err != nil {
		return false
	}
	return isLocalOAuthHost(target.Hostname())
}

func requestHost(c *gin.Context) string {
	if forwardedHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
		host, _, err := net.SplitHostPort(forwardedHost)
		if err == nil {
			return host
		}
		return forwardedHost
	}
	host, _, err := net.SplitHostPort(c.Request.Host)
	if err == nil {
		return host
	}
	return c.Request.Host
}

func isLocalOAuthHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return false
	}
	host = strings.TrimSuffix(host, ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return true
	}
	return false
}

func devOpenIDSeed(c *gin.Context) string {
	host := requestHost(c)
	if host == "" {
		host = "local"
	}
	return strings.NewReplacer(":", "-", ".", "-", "[", "", "]", "").Replace(host)
}

func (s *Server) MerchantApplicationStatus(c *gin.Context) {
	claims := currentClaims(c)
	user, err := s.userByID(claims.UserID)
	if err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	shop, _ := s.shopForMerchant(claims.UserID)
	application, _ := s.latestApplicationForUser(user)
	c.JSON(http.StatusOK, gin.H{
		"user":        user,
		"shop":        nullableShop(shop),
		"application": nullableApplication(application),
		"next_action": nextMerchantAction(shop, application),
	})
}

func (s *Server) UpdateMerchantApplication(c *gin.Context) {
	claims := currentClaims(c)
	user, err := s.userByID(claims.UserID)
	if err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var app models.MerchantApplication
	if err := s.DB.Where("id = ?", c.Param("id")).First(&app).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var merchant models.Merchant
	_ = s.DB.Where("user_id = ?", user.ID).First(&merchant).Error
	if app.ContactPhone != user.Phone && (app.MerchantID == nil || merchant.ID == 0 || *app.MerchantID != merchant.ID) {
		abort(c, http.StatusForbidden, errors.New("application does not belong to current merchant"))
		return
	}
	if app.Status == models.ApplicationApproved {
		abort(c, http.StatusBadRequest, errors.New("approved application cannot be edited"))
		return
	}
	var req createMerchantApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	app.ShopName = req.ShopName
	app.ContactName = req.ContactName
	app.ContactPhone = req.ContactPhone
	app.Category = req.Category
	app.PhotoURL = req.PhotoURL
	app.UsualArea = req.UsualArea
	app.Remark = req.Remark
	app.Status = models.ApplicationPending
	app.ReviewReason = ""
	app.ReviewerID = nil
	app.ReviewedAt = nil
	if app.ApplicationNo == "" {
		app.ApplicationNo = applicationNo()
	}
	if err := s.DB.Save(&app).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"application": app, "next_action": nextMerchantAction(models.Shop{}, app)})
}

func (s *Server) AdminListMerchantApplications(c *gin.Context) {
	var apps []models.MerchantApplication
	query := s.DB.Order("created_at desc")
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&apps).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"applications": apps})
}

func (s *Server) AdminGetMerchantApplication(c *gin.Context) {
	var app models.MerchantApplication
	if err := s.DB.First(&app, c.Param("id")).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"application": app})
}

func (s *Server) AdminApproveMerchantApplication(c *gin.Context) {
	claims := currentClaims(c)
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	var approvedApp models.MerchantApplication
	var shop models.Shop
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var app models.MerchantApplication
		if err := tx.First(&app, c.Param("id")).Error; err != nil {
			return err
		}
		if app.Status == models.ApplicationApproved && app.ShopID != nil {
			return tx.First(&shop, *app.ShopID).Error
		}
		var user models.User
		if err := tx.Where("phone = ? AND role = ?", app.ContactPhone, models.RoleMerchant).
			Attrs(models.User{Role: models.RoleMerchant, Phone: app.ContactPhone, Nickname: app.ContactName}).
			FirstOrCreate(&user).Error; err != nil {
			return err
		}
		var merchant models.Merchant
		if err := tx.Where("user_id = ?", user.ID).
			Attrs(models.Merchant{UserID: user.ID, Phone: app.ContactPhone, DisplayName: app.ContactName}).
			FirstOrCreate(&merchant).Error; err != nil {
			return err
		}
		if app.ShopID != nil {
			if err := tx.First(&shop, *app.ShopID).Error; err != nil {
				return err
			}
		} else if err := tx.Where("merchant_id = ?", merchant.ID).Order("id asc").First(&shop).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			shop = models.Shop{
				MerchantID:     merchant.ID,
				ShopCode:       s.uniqueShopCode(tx),
				Name:           app.ShopName,
				Category:       app.Category,
				AvatarURL:      app.PhotoURL,
				ContactPhone:   app.ContactPhone,
				Status:         models.ShopStatusActive,
				VerifiedStatus: models.VerifyVerified,
			}
			if err := tx.Create(&shop).Error; err != nil {
				return err
			}
		}
		now := time.Now()
		app.Status = models.ApplicationApproved
		app.ReviewReason = strings.TrimSpace(req.Reason)
		app.ReviewerID = uintPtr(claims.UserID)
		app.ReviewedAt = &now
		app.MerchantID = &merchant.ID
		app.ShopID = &shop.ID
		if app.ApplicationNo == "" {
			app.ApplicationNo = applicationNo()
		}
		if err := tx.Model(&shop).Updates(map[string]any{
			"name":            app.ShopName,
			"category":        app.Category,
			"avatar_url":      app.PhotoURL,
			"contact_phone":   app.ContactPhone,
			"status":          models.ShopStatusActive,
			"verified_status": models.VerifyVerified,
			"disabled_reason": "",
		}).Error; err != nil {
			return err
		}
		if err := tx.Save(&app).Error; err != nil {
			return err
		}
		approvedApp = app
		return nil
	})
	if err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"application": approvedApp, "shop": shop})
}

func (s *Server) AdminNeedsInfoMerchantApplication(c *gin.Context) {
	s.reviewMerchantApplication(c, models.ApplicationNeedsInfo)
}

func (s *Server) AdminRejectMerchantApplication(c *gin.Context) {
	s.reviewMerchantApplication(c, models.ApplicationRejected)
}

func (s *Server) reviewMerchantApplication(c *gin.Context, status string) {
	claims := currentClaims(c)
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	var app models.MerchantApplication
	if err := s.DB.First(&app, c.Param("id")).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	now := time.Now()
	app.Status = status
	app.ReviewReason = req.Reason
	app.ReviewerID = uintPtr(claims.UserID)
	app.ReviewedAt = &now
	if app.ApplicationNo == "" {
		app.ApplicationNo = applicationNo()
	}
	if err := s.DB.Save(&app).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"application": app})
}

func (s *Server) validateSMSCode(phone, scene, code string) error {
	scene = normalizeSMSScene(scene)
	if phone == "" || scene == "" || code == "" {
		return errors.New("phone, scene and code are required")
	}
	if s.Config.GinMode != "release" && code == "123456" {
		return nil
	}
	var record models.SMSCode
	err := s.DB.Where("phone = ? AND scene = ? AND code = ? AND used_at IS NULL AND expires_at > ?", phone, scene, code, time.Now()).
		Order("created_at desc").First(&record).Error
	if err != nil {
		return errors.New("invalid or expired verification code")
	}
	now := time.Now()
	return s.DB.Model(&record).Update("used_at", &now).Error
}

func (s *Server) latestApplicationForUser(user models.User) (models.MerchantApplication, error) {
	var merchant models.Merchant
	var merchantID uint
	if err := s.DB.Where("user_id = ?", user.ID).First(&merchant).Error; err == nil {
		merchantID = merchant.ID
	}
	var app models.MerchantApplication
	query := s.DB.Order("created_at desc")
	if merchantID > 0 {
		query = query.Where("contact_phone = ? OR merchant_id = ?", user.Phone, merchantID)
	} else {
		query = query.Where("contact_phone = ?", user.Phone)
	}
	return app, query.First(&app).Error
}

func (s *Server) userByID(id uint) (models.User, error) {
	var user models.User
	return user, s.DB.First(&user, id).Error
}

func (s *Server) uniqueShopCode(tx *gorm.DB) string {
	for i := 0; i < 8; i++ {
		code := services.RandomCode("s", 4)
		var count int64
		tx.Model(&models.Shop{}).Where("shop_code = ?", code).Count(&count)
		if count == 0 {
			return code
		}
	}
	return services.RandomCode("s", 8)
}

func normalizeSMSScene(scene string) string {
	scene = strings.ToLower(strings.TrimSpace(scene))
	switch scene {
	case "merchant", "admin", "customer":
		return scene
	default:
		return ""
	}
}

func nextMerchantAction(shop models.Shop, app models.MerchantApplication) string {
	if shop.ID != 0 {
		if shop.Status == models.ShopStatusDisabled {
			return "disabled"
		}
		return "dashboard"
	}
	switch app.Status {
	case models.ApplicationPending, models.ApplicationReviewing:
		return "application_pending"
	case models.ApplicationNeedsInfo:
		return "application_needs_info"
	case models.ApplicationRejected:
		return "application_rejected"
	case models.ApplicationApproved:
		return "application_approved"
	default:
		return "apply"
	}
}

func applicationNo() string {
	return "MA" + time.Now().Format("20060102") + services.RandomDigits(4)
}

func nullableApplication(app models.MerchantApplication) any {
	if app.ID == 0 {
		return nil
	}
	return app
}

func buildRequestOrigin(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := c.Request.Host
	if forwardedHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func uintPtr(id uint) *uint {
	return &id
}
