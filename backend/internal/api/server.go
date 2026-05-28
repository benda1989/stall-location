package api

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gkk/stall-location/backend/internal/auth"
	"github.com/gkk/stall-location/backend/internal/config"
	"github.com/gkk/stall-location/backend/internal/models"
	"github.com/gkk/stall-location/backend/internal/services"
	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Server struct {
	DB     *gorm.DB
	Config config.Config
}

func NewRouter(conn *gorm.DB, cfg config.Config) *gin.Engine {
	gin.SetMode(cfg.GinMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL, "http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	s := &Server{DB: conn, Config: cfg}
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	api := r.Group("/api")
	{
		api.POST("/auth/login", s.UnifiedLogin)
		api.POST("/auth/sms/send", s.SendSMSCode)
		api.GET("/wechat/js-config", s.WeChatJSConfig)
		api.GET("/wechat/oauth/silent/start", s.WeChatSilentStart)
		api.GET("/wechat/oauth/callback", s.WeChatOAuthCallback)
		api.GET("/stalls/nearby", s.NearbyStalls)
		api.GET("/shops/:shopCode", s.GetShop)
		api.GET("/shops/:shopCode/products", s.ListPublicProducts)
		api.GET("/shops/:shopCode/stall-session", s.GetPublicStallSession)
		api.GET("/shops/:shopCode/map-state", s.ShopMapState)
		api.POST("/orders", s.CreateOrder)
		api.GET("/orders/:orderNo", s.GetOrder)
		api.GET("/orders/:orderNo/location", s.OrderLocation)
		api.POST("/orders/:orderNo/cancel", s.CancelOrder)
		customer := api.Group("/customer")
		{
			customer.Use(s.RequireRole(models.RoleCustomer))
			customer.GET("/orders", s.ListCustomerOrders)
		}
		api.POST("/merchant-applications", s.CreateMerchantApplication)
		api.POST("/feedback", s.CreateFeedback)

		merchant := api.Group("/merchant")
		{
			merchant.POST("/auth/login", s.MerchantLogin)
			secured := merchant.Group("")
			secured.Use(s.RequireRole(models.RoleMerchant))
			secured.GET("/applications/me", s.MerchantApplicationStatus)
			secured.PUT("/applications/:id", s.UpdateMerchantApplication)
			secured.GET("/dashboard", s.MerchantDashboard)
			secured.POST("/shops", s.CreateShop)
			secured.PUT("/shops/:id", s.UpdateShop)
			secured.POST("/stall-sessions/start", s.StartStallSession)
			secured.POST("/stall-sessions/:id/end", s.EndStallSession)
			secured.GET("/products", s.ListMerchantProducts)
			secured.POST("/products", s.CreateProduct)
			secured.PUT("/products/:id", s.UpdateProduct)
			secured.GET("/orders", s.ListMerchantOrders)
			secured.POST("/orders/:id/accept", s.AcceptOrder)
			secured.POST("/orders/:id/reject", s.RejectOrder)
			secured.POST("/orders/:id/prepare", s.PrepareOrder)
			secured.POST("/orders/:id/ready", s.ReadyOrder)
			secured.POST("/orders/:id/complete", s.CompleteOrder)
			secured.GET("/qrcode", s.GetQRCode)
		}

		admin := api.Group("/admin")
		{
			admin.POST("/auth/login", s.AdminLogin)
			secured := admin.Group("")
			secured.Use(s.RequireRole(models.RoleAdmin))
			secured.GET("/merchant-applications", s.AdminListMerchantApplications)
			secured.GET("/merchant-applications/:id", s.AdminGetMerchantApplication)
			secured.POST("/merchant-applications/:id/approve", s.AdminApproveMerchantApplication)
			secured.POST("/merchant-applications/:id/needs-info", s.AdminNeedsInfoMerchantApplication)
			secured.POST("/merchant-applications/:id/reject", s.AdminRejectMerchantApplication)
			secured.GET("/shops", s.AdminListShops)
			secured.GET("/orders", s.AdminListOrders)
			secured.GET("/feedback", s.AdminListFeedback)
			secured.PUT("/feedback/:id", s.AdminUpdateFeedback)
			secured.GET("/stall-sessions/active", s.AdminActiveSessions)
			secured.POST("/orders/:id/refund", s.AdminRefundOrder)
			secured.POST("/orders/:id/cancel", s.AdminCancelOrder)
			secured.POST("/shops/:id/disable", s.AdminDisableShop)
			secured.POST("/shops/:id/enable", s.AdminEnableShop)
			secured.GET("/system/roles", s.AdminListSystemRoles)
			secured.POST("/system/roles", s.AdminCreateSystemRole)
			secured.PUT("/system/roles/:id", s.AdminUpdateSystemRole)
			secured.GET("/system/users", s.AdminListSystemUsers)
			secured.POST("/system/users", s.AdminCreateSystemUser)
			secured.PUT("/system/users/:id", s.AdminUpdateSystemUser)
			secured.GET("/system/menus", s.AdminListSystemMenus)
			secured.POST("/system/menus", s.AdminCreateSystemMenu)
			secured.PUT("/system/menus/:id", s.AdminUpdateSystemMenu)
		}
	}
	return r
}

func (s *Server) WeChatJSConfig(c *gin.Context) {
	pageURL := c.Query("url")
	if pageURL == "" {
		abort(c, http.StatusBadRequest, errors.New("url is required"))
		return
	}
	if s.Config.WeChatAppID == "" || s.Config.WeChatTicket == "" {
		c.JSON(http.StatusOK, gin.H{"enabled": false, "reason": "WECHAT_APP_ID or WECHAT_JSAPI_TICKET is not configured"})
		return
	}
	nonce := services.RandomCode("n", 8)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	raw := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%s&url=%s", s.Config.WeChatTicket, nonce, timestamp, pageURL)
	sum := sha1.Sum([]byte(raw))
	c.JSON(http.StatusOK, gin.H{
		"enabled":   true,
		"app_id":    s.Config.WeChatAppID,
		"timestamp": timestamp,
		"nonce_str": nonce,
		"signature": hex.EncodeToString(sum[:]),
		"js_api_list": []string{
			"getLocation",
			"openLocation",
			"updateAppMessageShareData",
			"updateTimelineShareData",
		},
	})
}

func (s *Server) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := auth.FromAuthorizationHeader(c.GetHeader("Authorization"))
		if err != nil {
			abort(c, http.StatusUnauthorized, err)
			return
		}
		claims, err := auth.Parse(s.Config.TokenSecret, token)
		if err != nil {
			abort(c, http.StatusUnauthorized, err)
			return
		}
		if claims.Role != role {
			abort(c, http.StatusForbidden, fmt.Errorf("requires %s role", role))
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}

func currentClaims(c *gin.Context) auth.Claims {
	claims, _ := c.Get("claims")
	if typed, ok := claims.(auth.Claims); ok {
		return typed
	}
	return auth.Claims{}
}

func (s *Server) optionalCustomerID(c *gin.Context) (*uint, error) {
	header := c.GetHeader("Authorization")
	if header == "" {
		return nil, nil
	}
	token, err := auth.FromAuthorizationHeader(header)
	if err != nil {
		return nil, err
	}
	claims, err := auth.Parse(s.Config.TokenSecret, token)
	if err != nil {
		return nil, err
	}
	if claims.Role != models.RoleCustomer {
		return nil, fmt.Errorf("requires %s role", models.RoleCustomer)
	}
	return &claims.UserID, nil
}

func (s *Server) optionalClaims(c *gin.Context) (*auth.Claims, error) {
	header := c.GetHeader("Authorization")
	if header == "" {
		return nil, nil
	}
	token, err := auth.FromAuthorizationHeader(header)
	if err != nil {
		return nil, err
	}
	claims, err := auth.Parse(s.Config.TokenSecret, token)
	if err != nil {
		return nil, err
	}
	return &claims, nil
}

func (s *Server) shopForMerchant(userID uint) (models.Shop, error) {
	var merchant models.Merchant
	if err := s.DB.Where("user_id = ?", userID).First(&merchant).Error; err != nil {
		return models.Shop{}, err
	}
	var shop models.Shop
	err := s.DB.Where("merchant_id = ?", merchant.ID).Order("id asc").First(&shop).Error
	return shop, err
}

type feedbackRequest struct {
	Source       string `json:"source"`
	ContactName  string `json:"contact_name"`
	ContactPhone string `json:"contact_phone"`
	Description  string `json:"description"`
	ImageURL     string `json:"image_url"`
	PageURL      string `json:"page_url"`
}

func (s *Server) CreateFeedback(c *gin.Context) {
	var req feedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	req.Source = strings.TrimSpace(req.Source)
	req.ContactName = strings.TrimSpace(req.ContactName)
	req.ContactPhone = strings.TrimSpace(req.ContactPhone)
	req.Description = strings.TrimSpace(req.Description)
	req.ImageURL = strings.TrimSpace(req.ImageURL)
	req.PageURL = strings.TrimSpace(req.PageURL)
	if req.Source == "" {
		req.Source = models.FeedbackSourceCustomer
	}
	if req.Source != models.FeedbackSourceCustomer && req.Source != models.FeedbackSourceMerchant {
		abort(c, http.StatusBadRequest, errors.New("source must be customer or merchant"))
		return
	}
	if req.Description == "" || req.ContactPhone == "" {
		abort(c, http.StatusBadRequest, errors.New("description and contact_phone are required"))
		return
	}

	feedback := models.Feedback{
		Source:       req.Source,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		Description:  req.Description,
		ImageURL:     req.ImageURL,
		PageURL:      req.PageURL,
		Status:       models.FeedbackStatusPending,
	}
	claims, err := s.optionalClaims(c)
	if err != nil {
		abort(c, http.StatusUnauthorized, err)
		return
	}
	if claims != nil {
		if claims.Role != req.Source {
			abort(c, http.StatusForbidden, fmt.Errorf("feedback source must match %s role", claims.Role))
			return
		}
		feedback.UserID = &claims.UserID
		if claims.Role == models.RoleMerchant {
			if shop, err := s.shopForMerchant(claims.UserID); err == nil {
				feedback.ShopID = &shop.ID
			}
		}
	}
	if err := s.DB.Create(&feedback).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"feedback": feedback})
}

func (s *Server) GetShop(c *gin.Context) {
	shop, err := s.findShopByCode(c.Param("shopCode"))
	if err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	active, _ := s.activeSession(shop.ID)
	c.JSON(http.StatusOK, gin.H{"shop": shop, "stall_session": nullableSession(active)})
}

func (s *Server) ListPublicProducts(c *gin.Context) {
	shop, err := s.findShopByCode(c.Param("shopCode"))
	if err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var products []models.Product
	err = s.DB.Where("shop_id = ? AND status = ?", shop.ID, models.ProductStatusOnSale).
		Order("sort_order asc, id asc").Find(&products).Error
	if err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": products})
}

func (s *Server) GetPublicStallSession(c *gin.Context) {
	shop, err := s.findShopByCode(c.Param("shopCode"))
	if err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	session, err := s.activeSession(shop.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"stall_session": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"stall_session": session})
}

type createOrderRequest struct {
	ShopCode      string            `json:"shop_code" binding:"required"`
	CustomerName  string            `json:"customer_name" binding:"required"`
	CustomerPhone string            `json:"customer_phone" binding:"required"`
	PickupTime    *time.Time        `json:"pickup_time"`
	Remark        string            `json:"remark"`
	Items         []createOrderItem `json:"items" binding:"required"`
}

type createOrderItem struct {
	ProductID uint `json:"product_id" binding:"required"`
	Quantity  int  `json:"quantity" binding:"required"`
}

func (s *Server) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	if len(req.Items) == 0 {
		abort(c, http.StatusBadRequest, errors.New("items cannot be empty"))
		return
	}
	customerID, err := s.optionalCustomerID(c)
	if err != nil {
		abort(c, http.StatusUnauthorized, err)
		return
	}

	var created models.Order
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var shop models.Shop
		if err := tx.Where("shop_code = ? AND status = ?", req.ShopCode, models.ShopStatusActive).First(&shop).Error; err != nil {
			return err
		}
		var session models.StallSession
		if err := tx.Where("shop_id = ? AND status = ? AND expected_end_at > ?", shop.ID, models.StallStatusActive, time.Now()).Order("started_at desc").First(&session).Error; err != nil {
			return fmt.Errorf("shop is not currently accepting orders")
		}

		order := models.Order{
			OrderNo:          services.OrderNo(),
			ShopID:           shop.ID,
			StallSessionID:   session.ID,
			CustomerID:       customerID,
			CustomerName:     req.CustomerName,
			CustomerPhone:    req.CustomerPhone,
			PickupCode:       services.PickupCode(),
			PickupTime:       req.PickupTime,
			Status:           models.OrderPendingAccept,
			PaymentStatus:    models.PaymentUnpaid,
			Remark:           req.Remark,
			LocationSnapshot: locationSnapshot(session),
		}

		for _, item := range req.Items {
			if item.Quantity <= 0 {
				return fmt.Errorf("invalid quantity for product %d", item.ProductID)
			}
			var product models.Product
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND shop_id = ? AND status = ?", item.ProductID, shop.ID, models.ProductStatusOnSale).First(&product).Error; err != nil {
				return err
			}
			if product.Stock < item.Quantity {
				return fmt.Errorf("%s stock is not enough", product.Name)
			}
			product.Stock -= item.Quantity
			if err := tx.Save(&product).Error; err != nil {
				return err
			}
			subtotal := product.PriceCents * int64(item.Quantity)
			order.TotalAmountCents += subtotal
			order.Items = append(order.Items, models.OrderItem{
				ProductID:      product.ID,
				ProductName:    product.Name,
				UnitPriceCents: product.PriceCents,
				Quantity:       item.Quantity,
				SubtotalCents:  subtotal,
			})
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		created = order
		return tx.Preload("Items").Preload("Shop").First(&created, order.ID).Error
	})
	if err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"order": created})
}

func (s *Server) GetOrder(c *gin.Context) {
	var order models.Order
	if err := s.DB.Preload("Items").Preload("Shop").Where("order_no = ?", c.Param("orderNo")).First(&order).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": order})
}

func (s *Server) ListCustomerOrders(c *gin.Context) {
	claims := currentClaims(c)
	var orders []models.Order
	if err := s.DB.Preload("Items").Preload("Shop").
		Where("customer_id = ?", claims.UserID).
		Order("created_at desc").
		Limit(200).
		Find(&orders).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (s *Server) CancelOrder(c *gin.Context) {
	var order models.Order
	if err := s.DB.Preload("Items").Where("order_no = ?", c.Param("orderNo")).First(&order).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	if !canCancel(order.Status) {
		abort(c, http.StatusBadRequest, fmt.Errorf("order cannot be canceled from %s", order.Status))
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := releaseStock(tx, order.Items); err != nil {
			return err
		}
		return tx.Model(&order).Update("status", models.OrderCanceled).Error
	}); err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order_no": order.OrderNo, "status": models.OrderCanceled})
}

type createMerchantApplicationRequest struct {
	ShopName     string `json:"shop_name" binding:"required"`
	ContactName  string `json:"contact_name" binding:"required"`
	ContactPhone string `json:"contact_phone" binding:"required"`
	Category     string `json:"category" binding:"required"`
	PhotoURL     string `json:"photo_url" binding:"required"`
	UsualArea    string `json:"usual_area"`
	Remark       string `json:"remark"`
}

func (s *Server) CreateMerchantApplication(c *gin.Context) {
	var req createMerchantApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	application := models.MerchantApplication{
		ApplicationNo: applicationNo(),
		ShopName:      req.ShopName,
		ContactName:   req.ContactName,
		ContactPhone:  req.ContactPhone,
		Category:      req.Category,
		PhotoURL:      req.PhotoURL,
		UsualArea:     req.UsualArea,
		Remark:        req.Remark,
		Status:        models.ApplicationPending,
	}
	if err := s.DB.Create(&application).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"application": application})
}

func (s *Server) MerchantLogin(c *gin.Context) {
	var req authLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	resp, status, err := s.merchantLoginResponse(req.Phone, req.Code)
	if err != nil {
		abort(c, status, err)
		return
	}
	c.JSON(status, resp)
}

func (s *Server) AdminLogin(c *gin.Context) {
	var req authLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	resp, status, err := s.adminLoginResponse(req)
	if err != nil {
		abort(c, status, err)
		return
	}
	c.JSON(status, resp)
}

func (s *Server) MerchantDashboard(c *gin.Context) {
	claims := currentClaims(c)
	shop, err := s.shopForMerchant(claims.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"shop": nil, "today_orders": 0, "today_amount_cents": 0, "pending_orders": 0, "stall_session": nil})
		return
	}
	start := time.Now().Truncate(24 * time.Hour)
	var todayOrders int64
	var pendingOrders int64
	var total struct{ Sum int64 }
	s.DB.Model(&models.Order{}).Where("shop_id = ? AND created_at >= ?", shop.ID, start).Count(&todayOrders)
	s.DB.Model(&models.Order{}).Select("COALESCE(SUM(total_amount_cents), 0) as sum").Where("shop_id = ? AND created_at >= ? AND status <> ?", shop.ID, start, models.OrderCanceled).Scan(&total)
	s.DB.Model(&models.Order{}).Where("shop_id = ? AND status = ?", shop.ID, models.OrderPendingAccept).Count(&pendingOrders)
	session, _ := s.activeSession(shop.ID)
	c.JSON(http.StatusOK, gin.H{"shop": shop, "today_orders": todayOrders, "today_amount_cents": total.Sum, "pending_orders": pendingOrders, "stall_session": nullableSession(session)})
}

func (s *Server) CreateShop(c *gin.Context) {
	claims := currentClaims(c)
	var merchant models.Merchant
	if err := s.DB.Where("user_id = ?", claims.UserID).First(&merchant).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var req struct {
		Name         string `json:"name" binding:"required"`
		Category     string `json:"category" binding:"required"`
		ContactPhone string `json:"contact_phone"`
		Announcement string `json:"announcement"`
		AvatarURL    string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	shop := models.Shop{
		MerchantID:     merchant.ID,
		ShopCode:       services.RandomCode("s", 4),
		Name:           req.Name,
		Category:       req.Category,
		ContactPhone:   req.ContactPhone,
		Announcement:   req.Announcement,
		AvatarURL:      req.AvatarURL,
		Status:         models.ShopStatusActive,
		VerifiedStatus: models.VerifyUnverified,
	}
	if err := s.DB.Create(&shop).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"shop": shop})
}

func (s *Server) UpdateShop(c *gin.Context) {
	claims := currentClaims(c)
	shop, err := s.ensureMerchantShop(claims.UserID, c.Param("id"))
	if err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	allowed := map[string]bool{"name": true, "category": true, "contact_phone": true, "announcement": true, "avatar_url": true}
	updates := map[string]any{}
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if err := s.DB.Model(&shop).Updates(updates).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	s.DB.First(&shop, shop.ID)
	c.JSON(http.StatusOK, gin.H{"shop": shop})
}

func (s *Server) StartStallSession(c *gin.Context) {
	claims := currentClaims(c)
	shop, err := s.shopForMerchant(claims.UserID)
	if err != nil {
		abort(c, http.StatusBadRequest, errors.New("create a shop before starting stall"))
		return
	}
	var req struct {
		Lat              float64    `json:"lat" binding:"required"`
		Lng              float64    `json:"lng" binding:"required"`
		Address          string     `json:"address" binding:"required"`
		LocationAccuracy int        `json:"location_accuracy"`
		ExpectedEndAt    *time.Time `json:"expected_end_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	expectedEnd := time.Now().Add(6 * time.Hour)
	if req.ExpectedEndAt != nil {
		expectedEnd = *req.ExpectedEndAt
	}
	now := time.Now()
	var session models.StallSession
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.StallSession{}).Where("shop_id = ? AND status = ?", shop.ID, models.StallStatusActive).
			Updates(map[string]any{"status": models.StallStatusEnded, "ended_at": now}).Error; err != nil {
			return err
		}
		session = models.StallSession{ShopID: shop.ID, Status: models.StallStatusActive, Lat: req.Lat, Lng: req.Lng, Address: req.Address, LocationAccuracy: req.LocationAccuracy, StartedAt: now, ExpectedEndAt: expectedEnd}
		return tx.Create(&session).Error
	})
	if err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"stall_session": session})
}

func (s *Server) EndStallSession(c *gin.Context) {
	claims := currentClaims(c)
	shop, err := s.shopForMerchant(claims.UserID)
	if err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var session models.StallSession
	if err := s.DB.Where("id = ? AND shop_id = ?", c.Param("id"), shop.ID).First(&session).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	now := time.Now()
	session.Status = models.StallStatusEnded
	session.EndedAt = &now
	if err := s.DB.Save(&session).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"stall_session": session})
}

func (s *Server) ListMerchantProducts(c *gin.Context) {
	claims := currentClaims(c)
	shop, err := s.shopForMerchant(claims.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"products": []models.Product{}})
		return
	}
	var products []models.Product
	if err := s.DB.Where("shop_id = ?", shop.ID).Order("sort_order asc, id asc").Find(&products).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": products})
}

func (s *Server) CreateProduct(c *gin.Context) {
	claims := currentClaims(c)
	shop, err := s.shopForMerchant(claims.UserID)
	if err != nil {
		abort(c, http.StatusBadRequest, errors.New("create a shop first"))
		return
	}
	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	if req.ImageURL == "" {
		abort(c, http.StatusBadRequest, errors.New("image_url is required"))
		return
	}
	product := models.Product{ShopID: shop.ID, Name: req.Name, Description: req.Description, PriceCents: req.PriceCents, Stock: req.Stock, ImageURL: req.ImageURL, Status: defaultString(req.Status, models.ProductStatusOnSale), SortOrder: req.SortOrder}
	if err := s.DB.Create(&product).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"product": product})
}

type productRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents" binding:"required"`
	Stock       int    `json:"stock"`
	ImageURL    string `json:"image_url"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
}

func (s *Server) UpdateProduct(c *gin.Context) {
	claims := currentClaims(c)
	shop, err := s.shopForMerchant(claims.UserID)
	if err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	var product models.Product
	if err := s.DB.Where("id = ? AND shop_id = ?", c.Param("id"), shop.ID).First(&product).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	product.Name = req.Name
	product.Description = req.Description
	product.PriceCents = req.PriceCents
	product.Stock = req.Stock
	product.ImageURL = req.ImageURL
	product.Status = defaultString(req.Status, models.ProductStatusOnSale)
	product.SortOrder = req.SortOrder
	if err := s.DB.Save(&product).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": product})
}

func (s *Server) ListMerchantOrders(c *gin.Context) {
	claims := currentClaims(c)
	shop, err := s.shopForMerchant(claims.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"orders": []models.Order{}})
		return
	}
	var orders []models.Order
	query := s.DB.Preload("Items").Where("shop_id = ?", shop.ID)
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at desc").Find(&orders).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (s *Server) AcceptOrder(c *gin.Context) {
	s.transitionMerchantOrder(c, models.OrderPendingAccept, models.OrderAccepted)
}
func (s *Server) PrepareOrder(c *gin.Context) {
	s.transitionMerchantOrder(c, models.OrderAccepted, models.OrderPreparing)
}
func (s *Server) ReadyOrder(c *gin.Context) { s.transitionMerchantOrder(c, "", models.OrderReady) }
func (s *Server) CompleteOrder(c *gin.Context) {
	s.transitionMerchantOrder(c, models.OrderReady, models.OrderCompleted)
}

func (s *Server) RejectOrder(c *gin.Context) {
	s.merchantCancelLike(c, models.OrderRejected)
}

func (s *Server) GetQRCode(c *gin.Context) {
	claims := currentClaims(c)
	shop, err := s.shopForMerchant(claims.UserID)
	if err != nil {
		abort(c, http.StatusBadRequest, errors.New("create a shop first"))
		return
	}
	shareURL := s.merchantShareURL(shop)
	png, err := qrcode.Encode(shareURL, qrcode.Medium, 512)
	if err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": shareURL, "qr_data_url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), "shop": shop})
}

func (s *Server) merchantShareURL(shop models.Shop) string {
	base := strings.TrimRight(s.Config.BaseURL, "/")
	query := url.Values{}
	query.Set("favorite", "1")
	query.Set("merchantId", strconv.FormatUint(uint64(shop.MerchantID), 10))
	query.Set("shopId", strconv.FormatUint(uint64(shop.ID), 10))
	return fmt.Sprintf("%s/s/%s?%s", base, url.PathEscape(shop.ShopCode), query.Encode())
}

func (s *Server) AdminListShops(c *gin.Context) {
	var shops []models.Shop
	orderBy := "CASE WHEN verified_status IN ('pending', 'unverified') THEN 0 WHEN status = 'disabled' THEN 1 ELSE 2 END, updated_at desc, created_at desc"
	if err := s.DB.Order(orderBy).Find(&shops).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"shops": shops})
}

func (s *Server) AdminListOrders(c *gin.Context) {
	var orders []models.Order
	query := s.DB.Preload("Items").Preload("Shop")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if paymentStatus := c.Query("payment_status"); paymentStatus != "" {
		query = query.Where("payment_status = ?", paymentStatus)
	}
	if shopID := c.Query("shop_id"); shopID != "" {
		query = query.Where("shop_id = ?", shopID)
	}
	if err := query.Order("created_at desc").Limit(200).Find(&orders).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

type adminFeedbackUpdateRequest struct {
	Status string `json:"status" binding:"required"`
	Note   string `json:"note"`
}

func (s *Server) AdminListFeedback(c *gin.Context) {
	var feedback []models.Feedback
	query := s.DB.Preload("User").Preload("Shop")
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if source := strings.TrimSpace(c.Query("source")); source != "" {
		query = query.Where("source = ?", source)
	}
	if err := query.
		Order("CASE status WHEN 'pending' THEN 0 WHEN 'handling' THEN 1 WHEN 'resolved' THEN 2 ELSE 3 END, created_at desc").
		Limit(300).
		Find(&feedback).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"feedback": feedback})
}

func (s *Server) AdminUpdateFeedback(c *gin.Context) {
	var req adminFeedbackUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, err)
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	req.Note = strings.TrimSpace(req.Note)
	if !validFeedbackStatus(req.Status) {
		abort(c, http.StatusBadRequest, errors.New("invalid feedback status"))
		return
	}
	var feedback models.Feedback
	if err := s.DB.First(&feedback, c.Param("id")).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	claims := currentClaims(c)
	updates := map[string]any{
		"status":       req.Status,
		"handler_id":   claims.UserID,
		"handler_note": req.Note,
	}
	if req.Status == models.FeedbackStatusResolved || req.Status == models.FeedbackStatusClosed {
		now := time.Now()
		updates["handled_at"] = &now
	} else {
		updates["handled_at"] = nil
	}
	if err := s.DB.Model(&feedback).Updates(updates).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	if err := s.DB.Preload("User").Preload("Shop").First(&feedback, feedback.ID).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"feedback": feedback})
}

func validFeedbackStatus(status string) bool {
	switch status {
	case models.FeedbackStatusPending, models.FeedbackStatusHandling, models.FeedbackStatusResolved, models.FeedbackStatusClosed:
		return true
	default:
		return false
	}
}

func (s *Server) AdminActiveSessions(c *gin.Context) {
	var sessions []models.StallSession
	s.expireSessions()
	if err := s.DB.Preload("Shop").Where("status = ? AND expected_end_at > ?", models.StallStatusActive, time.Now()).Order("started_at desc").Find(&sessions).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"stall_sessions": sessions})
}

func (s *Server) AdminRefundOrder(c *gin.Context) {
	var order models.Order
	if err := s.DB.First(&order, c.Param("id")).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	if order.PaymentStatus == models.PaymentRefunded {
		c.JSON(http.StatusOK, gin.H{"order": order})
		return
	}
	if order.PaymentStatus == models.PaymentUnpaid {
		abort(c, http.StatusBadRequest, errors.New("unpaid order cannot be refunded"))
		return
	}
	if err := s.DB.Model(&order).Update("payment_status", models.PaymentRefunded).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	order.PaymentStatus = models.PaymentRefunded
	c.JSON(http.StatusOK, gin.H{"order": order})
}

func (s *Server) AdminCancelOrder(c *gin.Context) {
	var order models.Order
	if err := s.DB.Preload("Items").First(&order, c.Param("id")).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	if !canCancel(order.Status) {
		abort(c, http.StatusBadRequest, fmt.Errorf("order cannot be canceled from %s", order.Status))
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := releaseStock(tx, order.Items); err != nil {
			return err
		}
		return tx.Model(&order).Update("status", models.OrderCanceled).Error
	}); err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	order.Status = models.OrderCanceled
	c.JSON(http.StatusOK, gin.H{"order": order})
}

func (s *Server) AdminDisableShop(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	var shop models.Shop
	if err := s.DB.First(&shop, c.Param("id")).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	if err := s.DB.Model(&shop).Updates(map[string]any{"status": models.ShopStatusDisabled, "disabled_reason": req.Reason}).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	shop.Status = models.ShopStatusDisabled
	shop.DisabledReason = req.Reason
	c.JSON(http.StatusOK, gin.H{"shop": shop})
}

func (s *Server) AdminEnableShop(c *gin.Context) {
	var shop models.Shop
	if err := s.DB.First(&shop, c.Param("id")).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	if err := s.DB.Model(&shop).Updates(map[string]any{"status": models.ShopStatusActive, "disabled_reason": ""}).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	shop.Status = models.ShopStatusActive
	shop.DisabledReason = ""
	c.JSON(http.StatusOK, gin.H{"shop": shop})
}

func (s *Server) transitionMerchantOrder(c *gin.Context, from string, to string) {
	claims := currentClaims(c)
	shop, err := s.shopForMerchant(claims.UserID)
	if err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var order models.Order
	if err := s.DB.Where("id = ? AND shop_id = ?", c.Param("id"), shop.ID).First(&order).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	if from != "" && order.Status != from {
		abort(c, http.StatusBadRequest, fmt.Errorf("order is %s, not %s", order.Status, from))
		return
	}
	if to == models.OrderReady && order.Status != models.OrderAccepted && order.Status != models.OrderPreparing {
		abort(c, http.StatusBadRequest, fmt.Errorf("order is %s, cannot mark ready", order.Status))
		return
	}
	order.Status = to
	if err := s.DB.Save(&order).Error; err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": order})
}

func (s *Server) merchantCancelLike(c *gin.Context, to string) {
	claims := currentClaims(c)
	shop, err := s.shopForMerchant(claims.UserID)
	if err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	var order models.Order
	if err := s.DB.Preload("Items").Where("id = ? AND shop_id = ?", c.Param("id"), shop.ID).First(&order).Error; err != nil {
		abort(c, http.StatusNotFound, err)
		return
	}
	if !canCancel(order.Status) {
		abort(c, http.StatusBadRequest, fmt.Errorf("order cannot be changed from %s", order.Status))
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := releaseStock(tx, order.Items); err != nil {
			return err
		}
		return tx.Model(&order).Update("status", to).Error
	}); err != nil {
		abort(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order_no": order.OrderNo, "status": to})
}

func (s *Server) ensureMerchantShop(userID uint, id string) (models.Shop, error) {
	var merchant models.Merchant
	if err := s.DB.Where("user_id = ?", userID).First(&merchant).Error; err != nil {
		return models.Shop{}, err
	}
	var shop models.Shop
	return shop, s.DB.Where("id = ? AND merchant_id = ?", id, merchant.ID).First(&shop).Error
}

func (s *Server) findShopByCode(code string) (models.Shop, error) {
	var shop models.Shop
	return shop, s.DB.Where("shop_code = ? AND status = ?", code, models.ShopStatusActive).First(&shop).Error
}

func (s *Server) activeSession(shopID uint) (models.StallSession, error) {
	s.expireSessions()
	var session models.StallSession
	return session, s.DB.Where("shop_id = ? AND status = ? AND expected_end_at > ?", shopID, models.StallStatusActive, time.Now()).Order("started_at desc").First(&session).Error
}

func (s *Server) expireSessions() {
	now := time.Now()
	_ = s.DB.Model(&models.StallSession{}).Where("status = ? AND expected_end_at <= ?", models.StallStatusActive, now).Updates(map[string]any{"status": models.StallStatusExpired, "ended_at": now}).Error
}

func locationSnapshot(session models.StallSession) datatypes.JSON {
	payload, _ := json.Marshal(map[string]any{"lat": session.Lat, "lng": session.Lng, "address": session.Address, "location_accuracy": session.LocationAccuracy})
	return datatypes.JSON(payload)
}

func releaseStock(tx *gorm.DB, items []models.OrderItem) error {
	for _, item := range items {
		if err := tx.Model(&models.Product{}).Where("id = ?", item.ProductID).Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
			return err
		}
	}
	return nil
}

func canCancel(status string) bool {
	return status == models.OrderPendingAccept || status == models.OrderAccepted || status == models.OrderPreparing
}

func abort(c *gin.Context, code int, err error) {
	c.AbortWithStatusJSON(code, gin.H{"error": err.Error()})
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func last4(phone string) string {
	if len(phone) <= 4 {
		return phone
	}
	return phone[len(phone)-4:]
}

func nullableShop(shop models.Shop) any {
	if shop.ID == 0 {
		return nil
	}
	return shop
}

func nullableSession(session models.StallSession) any {
	if session.ID == 0 {
		return nil
	}
	return session
}

func idParam(c *gin.Context, name string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	return uint(id), err
}
