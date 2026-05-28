package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gkk/stall-location/backend/internal/api"
	"github.com/gkk/stall-location/backend/internal/config"
	"github.com/gkk/stall-location/backend/internal/db"
	"github.com/gkk/stall-location/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	conn, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(conn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.SeedDemoData(conn); err != nil {
		t.Fatalf("seed: %v", err)
	}
	cfg := config.Config{FrontendURL: "http://localhost:5173", BaseURL: "http://localhost:5173", TokenSecret: "test-secret", GinMode: gin.TestMode}
	return api.NewRouter(conn, cfg), conn
}

func TestPublicShopAndOrderFlow(t *testing.T) {
	router, conn := setupRouter(t)

	resp := perform(router, http.MethodGet, "/api/shops/demo", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("shop status = %d, body = %s", resp.Code, resp.Body.String())
	}
	resp = perform(router, http.MethodGet, "/api/wechat/js-config?url=http%3A%2F%2Flocalhost%3A5173%2Fs%2Fdemo", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("wechat config status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var product models.Product
	if err := conn.Where("shop_id = ?", 1).First(&product).Error; err != nil {
		t.Fatalf("find seeded product: %v", err)
	}
	before := product.Stock
	orderReq := map[string]any{
		"shop_code":      "demo",
		"customer_name":  "小王",
		"customer_phone": "13800000001",
		"remark":         "少放酱",
		"items": []map[string]any{{
			"product_id": product.ID,
			"quantity":   2,
		}},
	}
	resp = perform(router, http.MethodPost, "/api/orders", orderReq, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create order status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var created struct {
		Order models.Order `json:"order"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	if created.Order.OrderNo == "" || created.Order.PickupCode == "" || created.Order.TotalAmountCents <= 0 {
		t.Fatalf("order missing generated fields: %+v", created.Order)
	}
	var after models.Product
	_ = conn.First(&after, product.ID).Error
	if after.Stock != before-2 {
		t.Fatalf("stock after order = %d, want %d", after.Stock, before-2)
	}

	resp = perform(router, http.MethodGet, "/api/orders/"+created.Order.OrderNo, nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("get order status = %d, body = %s", resp.Code, resp.Body.String())
	}
	resp = perform(router, http.MethodPost, "/api/orders/"+created.Order.OrderNo+"/cancel", map[string]any{}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("cancel order status = %d, body = %s", resp.Code, resp.Body.String())
	}
	_ = conn.First(&after, product.ID).Error
	if after.Stock != before {
		t.Fatalf("stock after public cancel = %d, want restored %d", after.Stock, before)
	}

	resp = perform(router, http.MethodPost, "/api/auth/login", map[string]any{"role": models.RoleCustomer, "wechat_code": "order-list-test"}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("customer login status = %d, body = %s", resp.Code, resp.Body.String())
	}
	customerToken := tokenFrom(t, resp.Body.Bytes())
	resp = perform(router, http.MethodPost, "/api/orders", orderReq, customerToken)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create authenticated order status = %d, body = %s", resp.Code, resp.Body.String())
	}
	resp = perform(router, http.MethodGet, "/api/customer/orders", nil, customerToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("customer order list status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var customerOrders struct {
		Orders []models.Order `json:"orders"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &customerOrders); err != nil {
		t.Fatalf("decode customer orders: %v", err)
	}
	if len(customerOrders.Orders) != 1 || customerOrders.Orders[0].CustomerID == nil || *customerOrders.Orders[0].CustomerID == 0 || len(customerOrders.Orders[0].Items) == 0 {
		t.Fatalf("customer order list should include the authenticated preorder with items: %+v", customerOrders.Orders)
	}
	resp = perform(router, http.MethodGet, "/api/customer/orders", nil, "")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized customer order list status = %d, body = %s", resp.Code, resp.Body.String())
	}

	resp = perform(router, http.MethodPost, "/api/merchant-applications", map[string]any{
		"shop_name":     "测试流动摊",
		"contact_name":  "李老板",
		"contact_phone": "13800000009",
		"category":      "早餐小吃",
		"photo_url":     "data:image/png;base64,ZmFrZQ==",
		"usual_area":    "地铁口附近",
		"remark":        "工作日早高峰出摊",
	}, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create merchant application status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var appResp struct {
		Application models.MerchantApplication `json:"application"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &appResp); err != nil {
		t.Fatalf("decode merchant application: %v", err)
	}
	if appResp.Application.ID == 0 || appResp.Application.Status != models.ApplicationPending {
		t.Fatalf("application missing fields: %+v", appResp.Application)
	}

	resp = perform(router, http.MethodPost, "/api/feedback", map[string]any{
		"source":        models.FeedbackSourceCustomer,
		"contact_name":  "小王",
		"contact_phone": "13800000001",
		"description":   "地图点位和实际位置有偏差",
		"image_url":     "data:image/png;base64,ZmFrZQ==",
		"page_url":      "/nearby",
	}, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create customer feedback status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var feedbackResp struct {
		Feedback models.Feedback `json:"feedback"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &feedbackResp); err != nil {
		t.Fatalf("decode customer feedback: %v", err)
	}
	if feedbackResp.Feedback.ID == 0 || feedbackResp.Feedback.Source != models.FeedbackSourceCustomer || feedbackResp.Feedback.Description == "" {
		t.Fatalf("feedback missing fields: %+v", feedbackResp.Feedback)
	}
}

func TestAdminBackofficeOrderAndShopActions(t *testing.T) {
	router, conn := setupRouter(t)

	resp := perform(router, http.MethodPost, "/api/admin/auth/login", map[string]any{"username": "admin", "password": "admin123"}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("admin login status = %d, body = %s", resp.Code, resp.Body.String())
	}
	adminToken := tokenFrom(t, resp.Body.Bytes())

	resp = perform(router, http.MethodGet, "/api/admin/shops", nil, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("admin shops status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var shopsResp struct {
		Shops []models.Shop `json:"shops"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &shopsResp); err != nil {
		t.Fatalf("decode shops: %v", err)
	}
	if len(shopsResp.Shops) < 4 || shopsResp.Shops[0].Status != models.ShopStatusDisabled {
		t.Fatalf("disabled/actionable shop should be sorted first: %+v", shopsResp.Shops)
	}

	var demoShop models.Shop
	if err := conn.Where("shop_code = ?", "demo").First(&demoShop).Error; err != nil {
		t.Fatalf("find demo shop: %v", err)
	}
	resp = perform(router, http.MethodPost, "/api/admin/shops/"+strconv.Itoa(int(demoShop.ID))+"/disable", map[string]any{"reason": "后台测试禁用"}, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("disable shop status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var changedShop models.Shop
	if err := conn.First(&changedShop, demoShop.ID).Error; err != nil {
		t.Fatalf("reload disabled shop: %v", err)
	}
	if changedShop.Status != models.ShopStatusDisabled || changedShop.DisabledReason == "" {
		t.Fatalf("shop not disabled with reason: %+v", changedShop)
	}
	resp = perform(router, http.MethodPost, "/api/admin/shops/"+strconv.Itoa(int(demoShop.ID))+"/enable", map[string]any{}, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("enable shop status = %d, body = %s", resp.Code, resp.Body.String())
	}
	if err := conn.First(&changedShop, demoShop.ID).Error; err != nil {
		t.Fatalf("reload enabled shop: %v", err)
	}
	if changedShop.Status != models.ShopStatusActive || changedShop.DisabledReason != "" {
		t.Fatalf("shop not enabled cleanly: %+v", changedShop)
	}

	resp = perform(router, http.MethodGet, "/api/admin/orders", nil, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("admin orders status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var ordersResp struct {
		Orders []models.Order `json:"orders"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &ordersResp); err != nil {
		t.Fatalf("decode orders: %v", err)
	}
	if len(ordersResp.Orders) < 5 || ordersResp.Orders[0].OrderNo != "DEMO-ORDER-PENDING" {
		t.Fatalf("admin orders should be newest first, got first=%+v", ordersResp.Orders)
	}

	var paidOrder models.Order
	if err := conn.Where("order_no = ?", "DEMO-ORDER-ACCEPTED").First(&paidOrder).Error; err != nil {
		t.Fatalf("find paid order: %v", err)
	}
	resp = perform(router, http.MethodPost, "/api/admin/orders/"+strconv.Itoa(int(paidOrder.ID))+"/refund", map[string]any{}, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("refund order status = %d, body = %s", resp.Code, resp.Body.String())
	}
	if err := conn.First(&paidOrder, paidOrder.ID).Error; err != nil {
		t.Fatalf("reload refunded order: %v", err)
	}
	if paidOrder.PaymentStatus != models.PaymentRefunded {
		t.Fatalf("payment status = %s, want refunded", paidOrder.PaymentStatus)
	}

	var product models.Product
	if err := conn.Where("shop_id = ?", demoShop.ID).First(&product).Error; err != nil {
		t.Fatalf("find product: %v", err)
	}
	before := product.Stock
	resp = perform(router, http.MethodPost, "/api/orders", map[string]any{
		"shop_code":      "demo",
		"customer_name":  "后台撤销顾客",
		"customer_phone": "13800000003",
		"items": []map[string]any{{
			"product_id": product.ID,
			"quantity":   2,
		}},
	}, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create order for admin cancel status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var created struct {
		Order models.Order `json:"order"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created cancel target: %v", err)
	}
	resp = perform(router, http.MethodPost, "/api/admin/orders/"+strconv.Itoa(int(created.Order.ID))+"/cancel", map[string]any{}, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("admin cancel order status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var canceled models.Order
	if err := conn.First(&canceled, created.Order.ID).Error; err != nil {
		t.Fatalf("reload canceled order: %v", err)
	}
	if canceled.Status != models.OrderCanceled {
		t.Fatalf("order status = %s, want canceled", canceled.Status)
	}
	if err := conn.First(&product, product.ID).Error; err != nil {
		t.Fatalf("reload product after admin cancel: %v", err)
	}
	if product.Stock != before {
		t.Fatalf("stock after admin cancel = %d, want restored %d", product.Stock, before)
	}

	resp = perform(router, http.MethodGet, "/api/admin/feedback", nil, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("admin feedback status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var feedbackResp struct {
		Feedback []models.Feedback `json:"feedback"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &feedbackResp); err != nil {
		t.Fatalf("decode admin feedback: %v", err)
	}
	if len(feedbackResp.Feedback) < 2 || feedbackResp.Feedback[0].Status != models.FeedbackStatusPending {
		t.Fatalf("pending feedback should be sorted first: %+v", feedbackResp.Feedback)
	}
	resp = perform(router, http.MethodPut, "/api/admin/feedback/"+strconv.Itoa(int(feedbackResp.Feedback[0].ID)), map[string]any{"status": models.FeedbackStatusResolved, "note": "已电话沟通并修正点位"}, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("update feedback status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var updatedFeedback struct {
		Feedback models.Feedback `json:"feedback"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &updatedFeedback); err != nil {
		t.Fatalf("decode updated feedback: %v", err)
	}
	if updatedFeedback.Feedback.Status != models.FeedbackStatusResolved || updatedFeedback.Feedback.HandlerNote == "" || updatedFeedback.Feedback.HandledAt == nil {
		t.Fatalf("feedback should be resolved with handler note/time: %+v", updatedFeedback.Feedback)
	}
}

func TestAdminSystemManagementAPIs(t *testing.T) {
	router, _ := setupRouter(t)

	resp := perform(router, http.MethodPost, "/api/admin/auth/login", map[string]any{"username": "admin", "password": "admin123"}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("admin login status = %d, body = %s", resp.Code, resp.Body.String())
	}
	adminToken := tokenFrom(t, resp.Body.Bytes())

	resp = perform(router, http.MethodGet, "/api/admin/system/roles", nil, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("system roles status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var rolesResp struct {
		Roles []models.SystemRole `json:"roles"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &rolesResp); err != nil {
		t.Fatalf("decode system roles: %v", err)
	}
	if len(rolesResp.Roles) < 3 || rolesResp.Roles[0].Code != "super_admin" || len(rolesResp.Roles[0].Menus) == 0 {
		t.Fatalf("seeded system roles missing menus: %+v", rolesResp.Roles)
	}

	resp = perform(router, http.MethodGet, "/api/admin/system/menus", nil, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("system menus status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var menusResp struct {
		Menus []models.SystemMenu `json:"menus"`
		Tree  []models.SystemMenu `json:"tree"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &menusResp); err != nil {
		t.Fatalf("decode system menus: %v", err)
	}
	if len(menusResp.Menus) < 8 || len(menusResp.Tree) == 0 {
		t.Fatalf("seeded system menus missing tree: %+v", menusResp)
	}

	resp = perform(router, http.MethodPost, "/api/admin/system/roles", map[string]any{
		"code":        "test_auditor",
		"name":        "测试审计员",
		"description": "只读审计",
		"status":      models.UserStatusActive,
		"menu_ids":    []uint{menusResp.Menus[0].ID},
	}, adminToken)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create system role status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var createdRole struct {
		Role models.SystemRole `json:"role"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &createdRole); err != nil {
		t.Fatalf("decode created role: %v", err)
	}
	if createdRole.Role.ID == 0 || len(createdRole.Role.Menus) != 1 {
		t.Fatalf("created role missing menu binding: %+v", createdRole.Role)
	}

	resp = perform(router, http.MethodPost, "/api/admin/system/users", map[string]any{
		"phone":    "13800130000",
		"nickname": "测试后台用户",
		"status":   models.UserStatusActive,
		"role_ids": []uint{createdRole.Role.ID},
	}, adminToken)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create system user status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var createdUser struct {
		User struct {
			models.User
			Roles []models.SystemRole `json:"roles"`
		} `json:"user"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &createdUser); err != nil {
		t.Fatalf("decode created system user: %v", err)
	}
	if createdUser.User.ID == 0 || len(createdUser.User.Roles) != 1 {
		t.Fatalf("created user missing role binding: %+v", createdUser.User)
	}

	resp = perform(router, http.MethodPut, "/api/admin/system/users/"+strconv.Itoa(int(createdUser.User.ID)), map[string]any{
		"phone":    "13800130000",
		"nickname": "测试后台用户",
		"status":   models.UserStatusDisabled,
		"role_ids": []uint{createdRole.Role.ID},
	}, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("disable system user status = %d, body = %s", resp.Code, resp.Body.String())
	}
	resp = perform(router, http.MethodPost, "/api/admin/auth/login", map[string]any{"phone": "13800130000", "code": "123456"}, "")
	if resp.Code != http.StatusForbidden {
		t.Fatalf("disabled system user login status = %d, want 403, body = %s", resp.Code, resp.Body.String())
	}
}

func TestUnifiedLoginEndpointIssuesJWTForAllRoles(t *testing.T) {
	router, _ := setupRouter(t)

	cases := []struct {
		name string
		body map[string]any
		role string
	}{
		{
			name: "customer",
			body: map[string]any{"role": models.RoleCustomer, "wechat_code": "unit-test"},
			role: models.RoleCustomer,
		},
		{
			name: "merchant",
			body: map[string]any{"role": models.RoleMerchant, "phone": "13800138000", "code": "123456"},
			role: models.RoleMerchant,
		},
		{
			name: "admin",
			body: map[string]any{"role": models.RoleAdmin, "phone": "13800139999", "code": "123456"},
			role: models.RoleAdmin,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := perform(router, http.MethodPost, "/api/auth/login", tc.body, "")
			if resp.Code != http.StatusOK {
				t.Fatalf("unified login status = %d, body = %s", resp.Code, resp.Body.String())
			}
			_ = tokenFrom(t, resp.Body.Bytes())
			var out struct {
				Role string `json:"role"`
			}
			if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
				t.Fatalf("decode role: %v", err)
			}
			if out.Role != tc.role {
				t.Fatalf("role = %s, want %s", out.Role, tc.role)
			}
		})
	}
}

func TestMerchantAndAdminAPIs(t *testing.T) {
	router, conn := setupRouter(t)

	resp := perform(router, http.MethodPost, "/api/merchant/auth/login", map[string]any{"phone": "13800138000", "code": "123456"}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("merchant login status = %d, body = %s", resp.Code, resp.Body.String())
	}
	merchantToken := tokenFrom(t, resp.Body.Bytes())
	resp = perform(router, http.MethodGet, "/api/merchant/dashboard", nil, merchantToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, body = %s", resp.Code, resp.Body.String())
	}
	resp = perform(router, http.MethodGet, "/api/merchant/qrcode", nil, merchantToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("qrcode status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var qrResp struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &qrResp); err != nil {
		t.Fatalf("decode qrcode: %v", err)
	}
	shareURL, err := url.Parse(qrResp.URL)
	if err != nil {
		t.Fatalf("parse qrcode url %q: %v", qrResp.URL, err)
	}
	if shareURL.Path != "/s/demo" || shareURL.Query().Get("favorite") != "1" || shareURL.Query().Get("merchantId") == "" || shareURL.Query().Get("shopId") == "" {
		t.Fatalf("qrcode url should carry shop and merchant identifiers for auto favorite: %s", qrResp.URL)
	}
	resp = perform(router, http.MethodGet, "/api/merchant/products", nil, merchantToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("products status = %d, body = %s", resp.Code, resp.Body.String())
	}
	resp = perform(router, http.MethodPost, "/api/merchant/products", map[string]any{
		"name":        "缺图新品",
		"price_cents": 900,
		"stock":       3,
		"status":      models.ProductStatusOnSale,
	}, merchantToken)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("create product without image status = %d, want 400, body = %s", resp.Code, resp.Body.String())
	}
	resp = perform(router, http.MethodPost, "/api/merchant/products", map[string]any{
		"name":        "测试新品",
		"description": "即取",
		"price_cents": 900,
		"stock":       3,
		"image_url":   "data:image/png;base64,ZmFrZQ==",
		"status":      models.ProductStatusOnSale,
	}, merchantToken)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create product with image status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var product models.Product
	if err := conn.Where("shop_id = ?", 1).First(&product).Error; err != nil {
		t.Fatalf("find product for merchant order flow: %v", err)
	}
	resp = perform(router, http.MethodPost, "/api/orders", map[string]any{
		"shop_code":      "demo",
		"customer_name":  "接口顾客",
		"customer_phone": "13800000002",
		"items": []map[string]any{{
			"product_id": product.ID,
			"quantity":   1,
		}},
	}, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create merchant-flow order status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var created struct {
		Order models.Order `json:"order"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode merchant-flow order: %v", err)
	}
	for _, step := range []string{"accept", "prepare", "ready", "complete"} {
		resp = perform(router, http.MethodPost, "/api/merchant/orders/"+strconv.Itoa(int(created.Order.ID))+"/"+step, map[string]any{}, merchantToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("order %s status = %d, body = %s", step, resp.Code, resp.Body.String())
		}
	}

	resp = perform(router, http.MethodPost, "/api/admin/auth/login", map[string]any{"username": "admin", "password": "admin123"}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("admin login status = %d, body = %s", resp.Code, resp.Body.String())
	}
	adminToken := tokenFrom(t, resp.Body.Bytes())
	resp = perform(router, http.MethodGet, "/api/admin/stall-sessions/active", nil, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("active sessions status = %d, body = %s", resp.Code, resp.Body.String())
	}
	resp = perform(router, http.MethodPost, "/api/feedback", map[string]any{
		"source":        models.FeedbackSourceMerchant,
		"contact_name":  "阿强",
		"contact_phone": "13800138000",
		"description":   "商户端希望增加批量改库存",
		"image_url":     "data:image/png;base64,ZmFrZQ==",
		"page_url":      "/merchant/dashboard",
	}, merchantToken)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create merchant feedback status = %d, body = %s", resp.Code, resp.Body.String())
	}

	resp = perform(router, http.MethodPost, "/api/auth/sms/send", map[string]any{"phone": "13800008888", "scene": "merchant"}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("send sms status = %d, body = %s", resp.Code, resp.Body.String())
	}
	resp = perform(router, http.MethodPost, "/api/merchant-applications", map[string]any{
		"shop_name":     "审核测试摊",
		"contact_name":  "赵老板",
		"contact_phone": "13800008888",
		"category":      "咖啡饮品",
		"photo_url":     "data:image/png;base64,ZmFrZQ==",
		"usual_area":    "园区东门",
	}, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create review application status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var appCreated struct {
		Application models.MerchantApplication `json:"application"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &appCreated); err != nil {
		t.Fatalf("decode review application: %v", err)
	}
	if appCreated.Application.ApplicationNo == "" {
		t.Fatalf("application should have application_no: %+v", appCreated.Application)
	}

	resp = perform(router, http.MethodGet, "/api/admin/merchant-applications", nil, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("list applications status = %d, body = %s", resp.Code, resp.Body.String())
	}
	resp = perform(router, http.MethodPost, "/api/admin/merchant-applications/"+strconv.Itoa(int(appCreated.Application.ID))+"/needs-info", map[string]any{"reason": "请补充清晰摊位照片"}, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("needs-info status = %d, body = %s", resp.Code, resp.Body.String())
	}

	resp = perform(router, http.MethodPost, "/api/merchant/auth/login", map[string]any{"phone": "13800008888", "code": "123456"}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("merchant login for pending app status = %d, body = %s", resp.Code, resp.Body.String())
	}
	reviewMerchantToken := tokenFrom(t, resp.Body.Bytes())
	var loginStatus struct {
		NextAction string `json:"next_action"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &loginStatus); err != nil {
		t.Fatalf("decode merchant next action: %v", err)
	}
	if loginStatus.NextAction != "application_needs_info" {
		t.Fatalf("next action = %s, want application_needs_info", loginStatus.NextAction)
	}

	resp = perform(router, http.MethodPut, "/api/merchant/applications/"+strconv.Itoa(int(appCreated.Application.ID)), map[string]any{
		"shop_name":     "审核测试摊",
		"contact_name":  "赵老板",
		"contact_phone": "13800008888",
		"category":      "咖啡饮品",
		"photo_url":     "data:image/png;base64,ZmFrZQ==",
		"usual_area":    "园区东门",
		"remark":        "补充了清晰照片",
	}, reviewMerchantToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("merchant update application status = %d, body = %s", resp.Code, resp.Body.String())
	}
	resp = perform(router, http.MethodPost, "/api/admin/merchant-applications/"+strconv.Itoa(int(appCreated.Application.ID))+"/approve", map[string]any{"reason": "资料完整"}, adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("approve application status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var approved struct {
		Application models.MerchantApplication `json:"application"`
		Shop        models.Shop                `json:"shop"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &approved); err != nil {
		t.Fatalf("decode approve response: %v", err)
	}
	if approved.Application.Status != models.ApplicationApproved || approved.Shop.ShopCode == "" {
		t.Fatalf("approve response missing fields: %+v", approved)
	}
	resp = perform(router, http.MethodPost, "/api/merchant/auth/login", map[string]any{"phone": "13800008888", "code": "123456"}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("approved merchant login status = %d, body = %s", resp.Code, resp.Body.String())
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &loginStatus); err != nil {
		t.Fatalf("decode approved merchant next action: %v", err)
	}
	if loginStatus.NextAction != "dashboard" {
		t.Fatalf("approved next action = %s, want dashboard", loginStatus.NextAction)
	}
}

func TestMapFirstAPIs(t *testing.T) {
	router, conn := setupRouter(t)

	resp := perform(router, http.MethodGet, "/api/stalls/nearby?lat=22.3193&lng=114.1694", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("nearby status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var nearby struct {
		EntryMode string `json:"entry_mode"`
		Stalls    []struct {
			Shop struct {
				ShopCode string `json:"shop_code"`
			} `json:"shop"`
			DistanceMeters *int `json:"distance_meters"`
		} `json:"stalls"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &nearby); err != nil {
		t.Fatalf("decode nearby: %v", err)
	}
	if nearby.EntryMode != "nearby" || len(nearby.Stalls) < 3 {
		t.Fatalf("nearby response should include multiple active stalls: %+v", nearby)
	}
	if nearby.Stalls[0].DistanceMeters == nil {
		t.Fatalf("nearby response should include distance when lat/lng provided")
	}

	resp = perform(router, http.MethodGet, "/api/stalls/nearby?min_lat=22.318&max_lat=22.320&min_lng=114.168&max_lng=114.170", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("nearby bounds status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var boundsNearby struct {
		LoadMode string `json:"load_mode"`
		Bounds   struct {
			MinLat float64 `json:"min_lat"`
			MaxLat float64 `json:"max_lat"`
		} `json:"bounds"`
		Stalls []struct {
			Shop struct {
				ShopCode string `json:"shop_code"`
			} `json:"shop"`
		} `json:"stalls"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &boundsNearby); err != nil {
		t.Fatalf("decode bounds nearby: %v", err)
	}
	if boundsNearby.LoadMode != "bounds" || len(boundsNearby.Stalls) != 1 || boundsNearby.Stalls[0].Shop.ShopCode != "demo" {
		t.Fatalf("bounds response should only include demo stall in viewport: %+v", boundsNearby)
	}

	resp = perform(router, http.MethodGet, "/api/stalls/nearby?category=%E5%92%96%E5%95%A1%E9%A5%AE%E5%93%81", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("nearby category status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var categoryNearby struct {
		Stalls []struct {
			Shop struct {
				ShopCode string `json:"shop_code"`
				Category string `json:"category"`
			} `json:"shop"`
		} `json:"stalls"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &categoryNearby); err != nil {
		t.Fatalf("decode category nearby: %v", err)
	}
	if len(categoryNearby.Stalls) != 1 || categoryNearby.Stalls[0].Shop.ShopCode != "coffee" || categoryNearby.Stalls[0].Shop.Category != "咖啡饮品" {
		t.Fatalf("category filter should only include coffee stall: %+v", categoryNearby.Stalls)
	}

	resp = perform(router, http.MethodGet, "/api/stalls/nearby?q=%E6%B0%B4%E6%9E%9C", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("nearby search status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var searchNearby struct {
		Stalls []struct {
			Shop struct {
				ShopCode string `json:"shop_code"`
			} `json:"shop"`
		} `json:"stalls"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &searchNearby); err != nil {
		t.Fatalf("decode search nearby: %v", err)
	}
	if len(searchNearby.Stalls) != 1 || searchNearby.Stalls[0].Shop.ShopCode != "fruit" {
		t.Fatalf("search query should only include matching fruit stall: %+v", searchNearby.Stalls)
	}

	resp = perform(router, http.MethodGet, "/api/shops/demo/map-state?lat=22.3193&lng=114.1694", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("map-state status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var focused struct {
		EntryMode string `json:"entry_mode"`
		Shop      struct {
			ShopCode string `json:"shop_code"`
		} `json:"shop"`
		DistanceMeters *int `json:"distance_meters"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &focused); err != nil {
		t.Fatalf("decode focused: %v", err)
	}
	if focused.EntryMode != "focused" || focused.Shop.ShopCode != "demo" || focused.DistanceMeters == nil {
		t.Fatalf("focused response should only describe target shop with distance: %+v", focused)
	}

	recentEnd := time.Now().Add(-24 * time.Hour)
	if err := conn.Model(&models.StallSession{}).Where("shop_id = ?", 1).Updates(map[string]any{"status": models.StallStatusEnded, "ended_at": recentEnd, "updated_at": recentEnd}).Error; err != nil {
		t.Fatalf("end demo stall: %v", err)
	}
	resp = perform(router, http.MethodGet, "/api/stalls/nearby?include_recent=1", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("nearby recent status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var recentNearby struct {
		Stalls []struct {
			Shop struct {
				ShopCode string `json:"shop_code"`
			} `json:"shop"`
			DisplayStatus string    `json:"display_status"`
			LastOnlineAt  time.Time `json:"last_online_at"`
		} `json:"stalls"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &recentNearby); err != nil {
		t.Fatalf("decode recent nearby: %v", err)
	}
	foundRecentDemo := false
	for _, stall := range recentNearby.Stalls {
		if stall.Shop.ShopCode == "demo" {
			foundRecentDemo = stall.DisplayStatus == "recent" && !stall.LastOnlineAt.IsZero()
		}
	}
	if !foundRecentDemo {
		t.Fatalf("recent search scope should include demo as recent inactive stall: %+v", recentNearby.Stalls)
	}

	resp = perform(router, http.MethodGet, "/api/shops/demo", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("shop after end status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var shopResp map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &shopResp); err != nil {
		t.Fatalf("decode shop after end: %v", err)
	}
	if shopResp["stall_session"] != nil {
		t.Fatalf("GetShop should return null stall_session when not active, got %v", shopResp["stall_session"])
	}
}

func perform(router *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func tokenFrom(t *testing.T, payload []byte) string {
	t.Helper()
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("decode token: %v", err)
	}
	if out.Token == "" {
		t.Fatalf("empty token in %s", string(payload))
	}
	if parts := strings.Split(out.Token, "."); len(parts) != 3 {
		t.Fatalf("token should be standard JWT with 3 parts, got %d parts: %s", len(parts), out.Token)
	}
	return out.Token
}
