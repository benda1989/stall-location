package db

import (
	"errors"
	"time"

	"github.com/gkk/stall-location/backend/internal/models"
	"gorm.io/gorm"
)

type demoShopSeed struct {
	Phone        string
	Nickname     string
	DisplayName  string
	ShopCode     string
	Name         string
	Category     string
	Announcement string
	Lat          float64
	Lng          float64
	Address      string
	Products     []demoProductSeed
}

type demoProductSeed struct {
	Name        string
	Description string
	PriceCents  int64
	Stock       int
}

func SeedDemoData(conn *gorm.DB) error {
	seeds := []demoShopSeed{
		{
			Phone:        "13800138000",
			Nickname:     "阿强煎饼",
			DisplayName:  "阿强",
			ShopCode:     "demo",
			Name:         "阿强流动煎饼铺",
			Category:     "早餐小吃",
			Announcement: "今天有薄脆煎饼、豆浆和茶叶蛋，支持预购自提。",
			Lat:          22.3193,
			Lng:          114.1694,
			Address:      "旺角地铁站 E2 口附近",
			Products: []demoProductSeed{
				{Name: "经典薄脆煎饼", Description: "鸡蛋、薄脆、葱花、秘制酱", PriceCents: 800, Stock: 30},
				{Name: "双蛋火腿煎饼", Description: "双蛋加火腿，适合早八赶路", PriceCents: 1200, Stock: 20},
				{Name: "现磨豆浆", Description: "微甜热豆浆", PriceCents: 300, Stock: 40},
				{Name: "茶叶蛋", Description: "入味老卤", PriceCents: 250, Stock: 36},
			},
		},
		{
			Phone:        "13800138001",
			Nickname:     "林姐咖啡车",
			DisplayName:  "林姐",
			ShopCode:     "coffee",
			Name:         "林姐移动咖啡车",
			Category:     "咖啡饮品",
			Announcement: "园区午后咖啡补给，冰美式和拿铁可提前点。",
			Lat:          22.3221,
			Lng:          114.1712,
			Address:      "创意园东门树荫下",
			Products: []demoProductSeed{
				{Name: "冰美式", Description: "深烘拼配，清爽提神", PriceCents: 1600, Stock: 24},
				{Name: "燕麦拿铁", Description: "燕麦奶，少糖默认", PriceCents: 2200, Stock: 18},
				{Name: "冷萃小瓶", Description: "限量冷萃，口感顺滑", PriceCents: 2600, Stock: 10},
			},
		},
		{
			Phone:        "13800138002",
			Nickname:     "老陈水果车",
			DisplayName:  "老陈",
			ShopCode:     "fruit",
			Name:         "老陈应季水果车",
			Category:     "水果鲜切",
			Announcement: "今日主推芒果杯、西瓜盒和冰镇椰青。",
			Lat:          22.3168,
			Lng:          114.1678,
			Address:      "社区南门便利店旁",
			Products: []demoProductSeed{
				{Name: "芒果鲜切杯", Description: "现切台芒，冰镇出杯", PriceCents: 1800, Stock: 16},
				{Name: "西瓜盒", Description: "无籽西瓜，适合下午茶", PriceCents: 1200, Stock: 22},
				{Name: "冰镇椰青", Description: "开盖即饮", PriceCents: 1500, Stock: 12},
			},
		},
	}
	for _, seed := range seeds {
		if err := ensureDemoShop(conn, seed); err != nil {
			return err
		}
	}
	if err := ensureAdmin(conn); err != nil {
		return err
	}
	if err := ensureSystemManagementData(conn); err != nil {
		return err
	}
	return ensureDemoBackofficeData(conn)
}

func ensureDemoShop(conn *gorm.DB, seed demoShopSeed) error {
	var shop models.Shop
	if err := conn.Where("shop_code = ?", seed.ShopCode).First(&shop).Error; err == nil {
		return ensureActiveSession(conn, shop.ID, seed.Lat, seed.Lng, seed.Address)
	}

	user := models.User{Role: models.RoleMerchant, Phone: seed.Phone, Nickname: seed.Nickname}
	if err := conn.Where("phone = ? AND role = ?", seed.Phone, models.RoleMerchant).FirstOrCreate(&user, user).Error; err != nil {
		return err
	}
	merchant := models.Merchant{UserID: user.ID, DisplayName: seed.DisplayName, Phone: seed.Phone}
	if err := conn.Where("user_id = ?", user.ID).FirstOrCreate(&merchant, merchant).Error; err != nil {
		return err
	}
	shop = models.Shop{
		MerchantID:     merchant.ID,
		ShopCode:       seed.ShopCode,
		Name:           seed.Name,
		Category:       seed.Category,
		Announcement:   seed.Announcement,
		ContactPhone:   seed.Phone,
		Status:         models.ShopStatusActive,
		VerifiedStatus: models.VerifyVerified,
	}
	if err := conn.Create(&shop).Error; err != nil {
		return err
	}
	products := make([]models.Product, 0, len(seed.Products))
	for i, product := range seed.Products {
		products = append(products, models.Product{
			ShopID:      shop.ID,
			Name:        product.Name,
			Description: product.Description,
			PriceCents:  product.PriceCents,
			Stock:       product.Stock,
			Status:      models.ProductStatusOnSale,
			SortOrder:   i + 1,
		})
	}
	if len(products) > 0 {
		if err := conn.Create(&products).Error; err != nil {
			return err
		}
	}
	return ensureActiveSession(conn, shop.ID, seed.Lat, seed.Lng, seed.Address)
}

func ensureActiveSession(conn *gorm.DB, shopID uint, lat float64, lng float64, address string) error {
	var session models.StallSession
	now := time.Now()
	if err := conn.Where("shop_id = ? AND status = ?", shopID, models.StallStatusActive).First(&session).Error; err == nil {
		return conn.Model(&session).Updates(map[string]any{
			"lat":             lat,
			"lng":             lng,
			"address":         address,
			"expected_end_at": now.Add(4 * time.Hour),
		}).Error
	}
	session = models.StallSession{ShopID: shopID, Status: models.StallStatusActive, Lat: lat, Lng: lng, Address: address, LocationAccuracy: 30, StartedAt: now.Add(-40 * time.Minute), ExpectedEndAt: now.Add(4 * time.Hour)}
	return conn.Create(&session).Error
}

func ensureAdmin(conn *gorm.DB) error {
	admin := models.User{Role: models.RoleAdmin, Phone: "admin", Nickname: "平台管理员"}
	if err := conn.Where("role = ? AND phone = ?", models.RoleAdmin, "admin").FirstOrCreate(&admin, admin).Error; err != nil {
		return err
	}
	if err := conn.Model(&admin).Where("status = '' OR status IS NULL").Update("status", models.UserStatusActive).Error; err != nil {
		return err
	}
	phoneAdmin := models.User{Role: models.RoleAdmin, Phone: "13800139999", Nickname: "平台管理员"}
	if err := conn.Where("role = ? AND phone = ?", models.RoleAdmin, phoneAdmin.Phone).FirstOrCreate(&phoneAdmin, phoneAdmin).Error; err != nil {
		return err
	}
	return conn.Model(&phoneAdmin).Where("status = '' OR status IS NULL").Update("status", models.UserStatusActive).Error
}

func ensureSystemManagementData(conn *gorm.DB) error {
	if err := conn.Model(&models.User{}).Where("status = '' OR status IS NULL").Update("status", models.UserStatusActive).Error; err != nil {
		return err
	}
	roles := []models.SystemRole{
		{Code: "super_admin", Name: "超级管理员", Description: "拥有系统、商户、订单和出摊全部权限", Status: models.UserStatusActive, SortOrder: 1},
		{Code: "ops_manager", Name: "运营经理", Description: "负责商户入驻、订单异常和出摊运营", Status: models.UserStatusActive, SortOrder: 2},
		{Code: "service_agent", Name: "客服专员", Description: "查看订单和商户资料，处理顾客沟通", Status: models.UserStatusActive, SortOrder: 3},
	}
	roleMap := map[string]models.SystemRole{}
	for _, seed := range roles {
		role := seed
		if err := conn.Where("code = ?", seed.Code).FirstOrCreate(&role, seed).Error; err != nil {
			return err
		}
		if err := conn.Model(&role).Updates(map[string]any{"name": seed.Name, "description": seed.Description, "status": seed.Status, "sort_order": seed.SortOrder}).Error; err != nil {
			return err
		}
		roleMap[seed.Code] = role
	}

	menus := []models.SystemMenu{
		{Code: "dashboard", Name: "经营看板", Path: "/admin", Icon: "dashboard", Type: "menu", Permission: "dashboard:view", Status: models.UserStatusActive, SortOrder: 1},
		{Code: "merchant_ops", Name: "商户管理", Path: "/admin?panel=merchants", Icon: "store", Type: "menu", Permission: "merchant:manage", Status: models.UserStatusActive, SortOrder: 2},
		{Code: "order_ops", Name: "订单管理", Path: "/admin?panel=orders", Icon: "orders", Type: "menu", Permission: "order:manage", Status: models.UserStatusActive, SortOrder: 3},
		{Code: "stall_map", Name: "出摊地图", Path: "/admin?panel=sessions", Icon: "map", Type: "menu", Permission: "stall:view", Status: models.UserStatusActive, SortOrder: 4},
		{Code: "system", Name: "系统管理", Path: "/admin?panel=system", Icon: "settings", Type: "directory", Permission: "system:view", Status: models.UserStatusActive, SortOrder: 10},
		{Code: "system.users", Name: "用户管理", Path: "/admin?panel=system&tab=users", Icon: "users", Type: "menu", Permission: "system:user:manage", Status: models.UserStatusActive, SortOrder: 11},
		{Code: "system.roles", Name: "角色管理", Path: "/admin?panel=system&tab=roles", Icon: "shield", Type: "menu", Permission: "system:role:manage", Status: models.UserStatusActive, SortOrder: 12},
		{Code: "system.menus", Name: "菜单管理", Path: "/admin?panel=system&tab=menus", Icon: "menu", Type: "menu", Permission: "system:menu:manage", Status: models.UserStatusActive, SortOrder: 13},
	}
	menuMap := map[string]models.SystemMenu{}
	for _, seed := range menus {
		menu := seed
		if seed.Code != "system" {
			if parent, ok := menuMap["system"]; ok && (seed.Code == "system.users" || seed.Code == "system.roles" || seed.Code == "system.menus") {
				menu.ParentID = &parent.ID
			}
		}
		if err := conn.Where("code = ?", seed.Code).FirstOrCreate(&menu, menu).Error; err != nil {
			return err
		}
		updates := map[string]any{"name": seed.Name, "path": seed.Path, "icon": seed.Icon, "type": seed.Type, "permission": seed.Permission, "status": seed.Status, "sort_order": seed.SortOrder}
		if menu.ParentID != nil {
			updates["parent_id"] = *menu.ParentID
		}
		if err := conn.Model(&menu).Updates(updates).Error; err != nil {
			return err
		}
		menuMap[seed.Code] = menu
	}

	if err := assignRoleMenus(conn, roleMap["super_admin"], allMenuIDs(menuMap)); err != nil {
		return err
	}
	if err := assignRoleMenus(conn, roleMap["ops_manager"], menuIDs(menuMap, "dashboard", "merchant_ops", "order_ops", "stall_map")); err != nil {
		return err
	}
	if err := assignRoleMenus(conn, roleMap["service_agent"], menuIDs(menuMap, "dashboard", "merchant_ops", "order_ops")); err != nil {
		return err
	}

	users := []struct {
		Phone    string
		Nickname string
		RoleCode string
		Status   string
	}{
		{Phone: "admin", Nickname: "平台管理员", RoleCode: "super_admin", Status: models.UserStatusActive},
		{Phone: "13800139999", Nickname: "平台管理员", RoleCode: "super_admin", Status: models.UserStatusActive},
		{Phone: "13800139998", Nickname: "运营经理", RoleCode: "ops_manager", Status: models.UserStatusActive},
		{Phone: "13800139997", Nickname: "客服专员", RoleCode: "service_agent", Status: models.UserStatusDisabled},
	}
	for _, seed := range users {
		user := models.User{Role: models.RoleAdmin, Phone: seed.Phone, Nickname: seed.Nickname, Status: seed.Status}
		if err := conn.Where("role = ? AND phone = ?", models.RoleAdmin, seed.Phone).FirstOrCreate(&user, user).Error; err != nil {
			return err
		}
		if err := conn.Model(&user).Updates(map[string]any{"nickname": seed.Nickname, "status": seed.Status}).Error; err != nil {
			return err
		}
		if role, ok := roleMap[seed.RoleCode]; ok {
			if err := assignUserRoles(conn, user.ID, []uint{role.ID}); err != nil {
				return err
			}
		}
	}
	return nil
}

func assignUserRoles(conn *gorm.DB, userID uint, roleIDs []uint) error {
	if err := conn.Where("user_id = ?", userID).Delete(&models.SystemUserRole{}).Error; err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if roleID == 0 {
			continue
		}
		if err := conn.Create(&models.SystemUserRole{UserID: userID, RoleID: roleID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func assignRoleMenus(conn *gorm.DB, role models.SystemRole, menuIDs []uint) error {
	if role.ID == 0 {
		return nil
	}
	if err := conn.Where("role_id = ?", role.ID).Delete(&models.SystemRoleMenu{}).Error; err != nil {
		return err
	}
	for _, menuID := range menuIDs {
		if menuID == 0 {
			continue
		}
		if err := conn.Create(&models.SystemRoleMenu{RoleID: role.ID, MenuID: menuID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func allMenuIDs(menus map[string]models.SystemMenu) []uint {
	ids := make([]uint, 0, len(menus))
	for _, menu := range menus {
		ids = append(ids, menu.ID)
	}
	return ids
}

func menuIDs(menus map[string]models.SystemMenu, codes ...string) []uint {
	ids := make([]uint, 0, len(codes))
	for _, code := range codes {
		if menu, ok := menus[code]; ok {
			ids = append(ids, menu.ID)
		}
	}
	return ids
}

func ensureDemoBackofficeData(conn *gorm.DB) error {
	if err := ensureDemoDisabledShop(conn); err != nil {
		return err
	}
	if err := ensureDemoApplications(conn); err != nil {
		return err
	}
	if err := ensureDemoOrders(conn); err != nil {
		return err
	}
	return ensureDemoFeedback(conn)
}

func ensureDemoDisabledShop(conn *gorm.DB) error {
	user := models.User{Role: models.RoleMerchant, Phone: "13800138003", Nickname: "周叔糖水"}
	if err := conn.Where("phone = ? AND role = ?", user.Phone, models.RoleMerchant).FirstOrCreate(&user, user).Error; err != nil {
		return err
	}
	merchant := models.Merchant{UserID: user.ID, DisplayName: "周叔", Phone: user.Phone}
	if err := conn.Where("user_id = ?", user.ID).FirstOrCreate(&merchant, merchant).Error; err != nil {
		return err
	}
	var shop models.Shop
	if err := conn.Where("shop_code = ?", "sweet").First(&shop).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		shop = models.Shop{
			MerchantID:     merchant.ID,
			ShopCode:       "sweet",
			Name:           "周叔糖水铺",
			Category:       "甜品冷饮",
			Announcement:   "后台演示：该商户当前被禁用。",
			ContactPhone:   user.Phone,
			Status:         models.ShopStatusDisabled,
			VerifiedStatus: models.VerifyVerified,
			DisabledReason: "证照信息待复核",
		}
		if err := conn.Create(&shop).Error; err != nil {
			return err
		}
	}
	return conn.Model(&shop).Updates(map[string]any{
		"category":        "甜品冷饮",
		"status":          models.ShopStatusDisabled,
		"verified_status": models.VerifyVerified,
		"disabled_reason": "证照信息待复核",
	}).Error
}

func ensureDemoApplications(conn *gorm.DB) error {
	now := time.Now()
	apps := []models.MerchantApplication{
		{
			ApplicationNo: "APP-DEMO-PENDING",
			ShopName:      "小敏饭团车",
			ContactName:   "小敏",
			ContactPhone:  "13800138110",
			Category:      "早餐小吃",
			PhotoURL:      "data:image/png;base64,ZmFrZQ==",
			UsualArea:     "写字楼北门",
			Remark:        "后台演示：待审核申请",
			Status:        models.ApplicationPending,
		},
		{
			ApplicationNo: "APP-DEMO-NEEDS",
			ShopName:      "阿海烤肠",
			ContactName:   "阿海",
			ContactPhone:  "13800138111",
			Category:      "夜宵小吃",
			PhotoURL:      "data:image/png;base64,ZmFrZQ==",
			UsualArea:     "社区西门",
			Remark:        "后台演示：需要补充资料",
			Status:        models.ApplicationNeedsInfo,
			ReviewReason:  "请补充摊位近照和常驻位置",
			ReviewedAt:    &now,
		},
		{
			ApplicationNo: "APP-DEMO-REJECTED",
			ShopName:      "临时无证摊",
			ContactName:   "测试商户",
			ContactPhone:  "13800138112",
			Category:      "其他",
			PhotoURL:      "data:image/png;base64,ZmFrZQ==",
			UsualArea:     "未知",
			Status:        models.ApplicationRejected,
			ReviewReason:  "经营范围与平台规则不匹配",
			ReviewedAt:    &now,
		},
	}
	for _, app := range apps {
		var existing models.MerchantApplication
		err := conn.Where("application_no = ?", app.ApplicationNo).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := conn.Create(&app).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureDemoOrders(conn *gorm.DB) error {
	var shop models.Shop
	if err := conn.Where("shop_code = ?", "demo").First(&shop).Error; err != nil {
		return err
	}
	var session models.StallSession
	if err := conn.Where("shop_id = ?", shop.ID).Order("started_at desc").First(&session).Error; err != nil {
		return err
	}
	var product models.Product
	if err := conn.Where("shop_id = ?", shop.ID).Order("sort_order asc, id asc").First(&product).Error; err != nil {
		return err
	}

	now := time.Now()
	orders := []struct {
		OrderNo       string
		Name          string
		Status        string
		PaymentStatus string
		Quantity      int
		CreatedAt     time.Time
	}{
		{"DEMO-ORDER-PENDING", "后台待接单顾客", models.OrderPendingAccept, models.PaymentUnpaid, 1, now.Add(-12 * time.Minute)},
		{"DEMO-ORDER-ACCEPTED", "后台已接单顾客", models.OrderAccepted, models.PaymentPaid, 2, now.Add(-35 * time.Minute)},
		{"DEMO-ORDER-READY", "后台待取餐顾客", models.OrderReady, models.PaymentPaid, 1, now.Add(-70 * time.Minute)},
		{"DEMO-ORDER-CANCELED", "后台已取消顾客", models.OrderCanceled, models.PaymentUnpaid, 1, now.Add(-2 * time.Hour)},
		{"DEMO-ORDER-REFUNDED", "后台已退款顾客", models.OrderCompleted, models.PaymentRefunded, 1, now.Add(-3 * time.Hour)},
	}
	for _, seed := range orders {
		var existing models.Order
		if err := conn.Where("order_no = ?", seed.OrderNo).First(&existing).Error; err == nil {
			continue
		}
		subtotal := product.PriceCents * int64(seed.Quantity)
		order := models.Order{
			OrderNo:          seed.OrderNo,
			ShopID:           shop.ID,
			StallSessionID:   session.ID,
			CustomerName:     seed.Name,
			CustomerPhone:    "13800139000",
			PickupCode:       "D" + seed.OrderNo[len(seed.OrderNo)-4:],
			Status:           seed.Status,
			PaymentStatus:    seed.PaymentStatus,
			TotalAmountCents: subtotal,
			Remark:           "后台演示订单",
			CreatedAt:        seed.CreatedAt,
			UpdatedAt:        seed.CreatedAt,
			Items: []models.OrderItem{{
				ProductID:      product.ID,
				ProductName:    product.Name,
				UnitPriceCents: product.PriceCents,
				Quantity:       seed.Quantity,
				SubtotalCents:  subtotal,
				CreatedAt:      seed.CreatedAt,
				UpdatedAt:      seed.CreatedAt,
			}},
		}
		if err := conn.Create(&order).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureDemoFeedback(conn *gorm.DB) error {
	var demoShop models.Shop
	_ = conn.Where("shop_code = ?", "demo").First(&demoShop).Error
	var coffeeShop models.Shop
	_ = conn.Where("shop_code = ?", "coffee").First(&coffeeShop).Error

	now := time.Now()
	items := []models.Feedback{
		{
			Source:       models.FeedbackSourceCustomer,
			ShopID:       uintPtrOrNil(demoShop.ID),
			ContactName:  "小王",
			ContactPhone: "13800139010",
			Description:  "地图显示在地铁口，实际摊位在路对面，建议调整点位提示。",
			PageURL:      "/nearby",
			Status:       models.FeedbackStatusPending,
			CreatedAt:    now.Add(-18 * time.Minute),
			UpdatedAt:    now.Add(-18 * time.Minute),
		},
		{
			Source:       models.FeedbackSourceMerchant,
			ShopID:       uintPtrOrNil(coffeeShop.ID),
			ContactName:  "林姐",
			ContactPhone: "13800138001",
			Description:  "商品库存改动后希望能批量保存，移动端单个点击有点慢。",
			PageURL:      "/merchant/products",
			Status:       models.FeedbackStatusHandling,
			HandlerNote:  "已记录到商品管理优化池。",
			CreatedAt:    now.Add(-52 * time.Minute),
			UpdatedAt:    now.Add(-20 * time.Minute),
		},
	}
	for _, item := range items {
		var existing models.Feedback
		err := conn.Where("source = ? AND contact_phone = ? AND description = ?", item.Source, item.ContactPhone, item.Description).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := conn.Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func uintPtrOrNil(value uint) *uint {
	if value == 0 {
		return nil
	}
	return &value
}
