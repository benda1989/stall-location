package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gkk/stall-location/backend/internal/auth"
	"github.com/gkk/stall-location/backend/internal/models"
	"gorm.io/gorm"
)

type authLoginRequest struct {
	Role       string `json:"role"`
	Phone      string `json:"phone"`
	Code       string `json:"code"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	OpenID     string `json:"openid"`
	DevOpenID  string `json:"dev_openid"`
	WeChatCode string `json:"wechat_code"`
}

func (s *Server) UnifiedLogin(c *gin.Context) {
	var req authLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	resp, status, err := s.loginByRequest(req)
	if err != nil {
		abort(c, status, err)
		return
	}
	c.JSON(status, resp)
}

func (s *Server) loginByRequest(req authLoginRequest) (gin.H, int, error) {
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role == "" && (req.Username != "" || req.Password != "") {
		role = models.RoleAdmin
	}
	switch role {
	case models.RoleCustomer:
		return s.customerLoginResponse(req)
	case models.RoleMerchant:
		return s.merchantLoginResponse(req.Phone, req.Code)
	case models.RoleAdmin:
		return s.adminLoginResponse(req)
	default:
		return nil, http.StatusBadRequest, errors.New("role must be customer, merchant or admin")
	}
}

func (s *Server) merchantLoginResponse(phone, code string) (gin.H, int, error) {
	phone = strings.TrimSpace(phone)
	code = strings.TrimSpace(code)
	if err := s.validateSMSCode(phone, models.RoleMerchant, code); err != nil {
		return nil, http.StatusUnauthorized, err
	}
	var user models.User
	if err := s.DB.Where("phone = ? AND role = ?", phone, models.RoleMerchant).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusInternalServerError, err
		}
		user = models.User{Phone: phone, Role: models.RoleMerchant, Nickname: "摊主" + last4(phone), Status: models.UserStatusActive}
		if err := s.DB.Create(&user).Error; err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}
	if user.Status == models.UserStatusDisabled {
		return nil, http.StatusForbidden, errors.New("merchant user is disabled")
	}
	var merchant models.Merchant
	if err := s.DB.Where("user_id = ?", user.ID).First(&merchant).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusInternalServerError, err
		}
		merchant = models.Merchant{UserID: user.ID, Phone: phone, DisplayName: user.Nickname}
		if err := s.DB.Create(&merchant).Error; err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}
	token, err := auth.Sign(s.Config.TokenSecret, auth.Claims{UserID: user.ID, Role: models.RoleMerchant})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	shop, _ := s.shopForMerchant(user.ID)
	application, _ := s.latestApplicationForUser(user)
	return gin.H{
		"token":       token,
		"role":        models.RoleMerchant,
		"user":        user,
		"merchant":    merchant,
		"shop":        nullableShop(shop),
		"application": nullableApplication(application),
		"next_action": nextMerchantAction(shop, application),
	}, http.StatusOK, nil
}

func (s *Server) adminLoginResponse(req authLoginRequest) (gin.H, int, error) {
	req.Phone = strings.TrimSpace(req.Phone)
	req.Code = strings.TrimSpace(req.Code)
	req.Username = strings.TrimSpace(req.Username)
	var user models.User
	if req.Phone != "" || req.Code != "" {
		if req.Phone == "" || req.Code == "" {
			return nil, http.StatusBadRequest, errors.New("phone and code are required")
		}
		if err := s.validateSMSCode(req.Phone, models.RoleAdmin, req.Code); err != nil {
			return nil, http.StatusUnauthorized, err
		}
		if err := s.DB.Where("role = ? AND phone = ?", models.RoleAdmin, req.Phone).First(&user).Error; err != nil {
			return nil, http.StatusForbidden, errors.New("phone has no admin permission")
		}
	} else {
		if req.Username != "admin" || req.Password != "admin123" {
			return nil, http.StatusUnauthorized, errors.New("invalid admin credentials")
		}
		if err := s.DB.Where("role = ? AND phone = ?", models.RoleAdmin, "admin").First(&user).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, http.StatusInternalServerError, err
			}
			user = models.User{Role: models.RoleAdmin, Phone: "admin", Nickname: "平台管理员", Status: models.UserStatusActive}
			if err := s.DB.Create(&user).Error; err != nil {
				return nil, http.StatusInternalServerError, err
			}
		}
	}
	if user.Status == models.UserStatusDisabled {
		return nil, http.StatusForbidden, errors.New("admin user is disabled")
	}
	token, err := auth.Sign(s.Config.TokenSecret, auth.Claims{UserID: user.ID, Role: models.RoleAdmin})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return gin.H{"token": token, "role": models.RoleAdmin, "user": user}, http.StatusOK, nil
}

func (s *Server) customerLoginResponse(req authLoginRequest) (gin.H, int, error) {
	openID := strings.TrimSpace(req.OpenID)
	if openID == "" {
		openID = strings.TrimSpace(req.DevOpenID)
	}
	if openID == "" && strings.TrimSpace(req.WeChatCode) != "" {
		var err error
		openID, err = s.openIDFromWeChatCode(context.Background(), req.WeChatCode)
		if err != nil {
			return nil, http.StatusBadGateway, err
		}
	}
	if openID == "" && strings.TrimSpace(req.Code) != "" && req.Phone == "" {
		var err error
		openID, err = s.openIDFromWeChatCode(context.Background(), req.Code)
		if err != nil {
			return nil, http.StatusBadGateway, err
		}
	}
	if openID == "" {
		return nil, http.StatusBadRequest, errors.New("openid or wechat_code is required")
	}
	var user models.User
	if err := s.DB.Where("role = ? AND open_id = ?", models.RoleCustomer, openID).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusInternalServerError, err
		}
		user = models.User{Role: models.RoleCustomer, OpenID: openID, Nickname: "微信顾客", Status: models.UserStatusActive}
		if err := s.DB.Create(&user).Error; err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}
	if user.Status == models.UserStatusDisabled {
		return nil, http.StatusForbidden, errors.New("customer user is disabled")
	}
	token, err := auth.Sign(s.Config.TokenSecret, auth.Claims{UserID: user.ID, Role: models.RoleCustomer})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return gin.H{"token": token, "role": models.RoleCustomer, "user": user, "openid": openID}, http.StatusOK, nil
}
