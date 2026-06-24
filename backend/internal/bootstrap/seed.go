package bootstrap

import (
	"errors"
	"fmt"
	"time"

	sysmodel "gkk/handler/urm/model"
	"gkk/model/tree"

	bizmodel "github.com/gkk/stall-location/backend/internal/model"
	"gorm.io/gorm"
)

func SeedDemoData(db *gorm.DB) error {
	if err := seedSysAdmin(db); err != nil {
		return err
	}
	if err := seedDemoMerchant(db); err != nil {
		return err
	}
	return nil
}

func seedDemoMerchant(db *gorm.DB) error {
	merchantUser := bizmodel.User{}
	if err := db.Where("phone = ?", "13800138000").FirstOrCreate(&merchantUser, merchantUser).Error; err != nil {
		return err
	}
	merchantUser.Phone = "13800138000"
	merchantUser.Username = "13800138000"
	merchantUser.Nickname = "阿强摊主"
	merchantUser.Password = merchantUser.SetPassword("123456")
	merchantUser.IsValid = true
	merchantUser.IsLock = false
	if err := db.Select("phone", "username", "nickname", "password", "is_valid", "is_lock").Save(&merchantUser).Error; err != nil {
		return err
	}

	merchant := bizmodel.Merchant{MerchantItem: bizmodel.MerchantItem{UserID: merchantUser.Id, DisplayName: "阿强流动煎饼铺"}}
	if err := db.Where("user_id = ?", merchantUser.Id).FirstOrCreate(&merchant, merchant).Error; err != nil {
		return err
	}
	merchant.UserID = merchantUser.Id
	merchant.DisplayName = "阿强流动煎饼铺"
	merchant.Phone = "13800138000"
	merchant.Category = "早餐小吃"
	merchant.AvatarURL = "https://images.unsplash.com/photo-1525351484163-7529414344d8?auto=format&fit=crop&w=800&q=80"
	merchant.Announcement = "现摊煎饼、豆浆，支持到摊自取。"
	merchant.ContactPhone = "13800138000"
	merchant.Status = bizmodel.StatusActive
	merchant.VerifyStatus = bizmodel.VerifyVerified
	merchant.DisabledReason = ""
	merchant.ShareCode = "8K3XQ2"
	if err := db.Select("user_id", "display_name", "phone", "category", "avatar_url", "announcement", "contact_phone", "status", "verify_status", "disabled_reason", "share_code").Save(&merchant).Error; err != nil {
		return err
	}
	if err := db.Model(&merchantUser).Update("merchant_id", merchant.Id).Error; err != nil {
		return err
	}

	products := []bizmodel.ProductItem{
		{MerchantID: merchant.Id, Name: "招牌煎饼", Description: "鸡蛋、薄脆、秘制酱", PriceCents: 1200, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1512058564366-18510be2db19?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 20},
		{MerchantID: merchant.Id, Name: "热豆浆", Description: "现磨热豆浆", PriceCents: 500, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1544145945-f90425340c7e?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 19},
		{MerchantID: merchant.Id, Name: "双蛋豪华煎饼", Description: "双蛋加肠，适合早餐", PriceCents: 1800, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1525351484163-7529414344d8?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 18},
		{MerchantID: merchant.Id, Name: "香葱鸡蛋饼", Description: "香葱、鸡蛋、薄脆", PriceCents: 1000, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1504674900247-0877df9cc836?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 17},
		{MerchantID: merchant.Id, Name: "里脊煎饼", Description: "煎里脊、鸡蛋、生菜", PriceCents: 1600, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1565299624946-b28f40a0ae38?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 16},
		{MerchantID: merchant.Id, Name: "火腿煎饼", Description: "火腿、薄脆、甜面酱", PriceCents: 1400, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1604908176997-125f25cc6f3d?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 15},
		{MerchantID: merchant.Id, Name: "培根煎饼", Description: "培根、鸡蛋、黑椒酱", PriceCents: 1700, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1525351484163-7529414344d8?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 14},
		{MerchantID: merchant.Id, Name: "素菜煎饼", Description: "生菜、土豆丝、脆饼", PriceCents: 900, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1546069901-ba9599a7e63c?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 13},
		{MerchantID: merchant.Id, Name: "辣酱煎饼", Description: "自制辣酱、双层薄脆", PriceCents: 1300, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1551183053-bf91a1d81141?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 12},
		{MerchantID: merchant.Id, Name: "黑椒鸡排煎饼", Description: "鸡排、黑椒、生菜", PriceCents: 1900, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1562967916-eb82221dfb36?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 11},
		{MerchantID: merchant.Id, Name: "芝士煎饼", Description: "芝士片、鸡蛋、薄脆", PriceCents: 1800, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1482049016688-2d3e1b311543?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 10},
		{MerchantID: merchant.Id, Name: "牛肉煎饼", Description: "酱牛肉、鸡蛋、香菜", PriceCents: 2200, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1550547660-d9450f859349?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 9},
		{MerchantID: merchant.Id, Name: "玉米煎饼", Description: "甜玉米、鸡蛋、沙拉酱", PriceCents: 1200, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1550317138-10000687a72b?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 8},
		{MerchantID: merchant.Id, Name: "土豆丝煎饼", Description: "炝拌土豆丝、薄脆", PriceCents: 1100, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1505576399279-565b52d4ac71?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 7},
		{MerchantID: merchant.Id, Name: "肉松煎饼", Description: "海苔肉松、鸡蛋", PriceCents: 1500, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1565958011703-44f9829ba187?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 6},
		{MerchantID: merchant.Id, Name: "豆腐脑", Description: "咸香卤汁，早餐搭配", PriceCents: 700, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1609501676725-7186f017a4b7?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 5},
		{MerchantID: merchant.Id, Name: "小米粥", Description: "慢熬小米粥", PriceCents: 600, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1517673132405-a56a62b18caf?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 4},
		{MerchantID: merchant.Id, Name: "茶叶蛋", Description: "五香入味", PriceCents: 300, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1498654896293-37aacf113fd9?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 3},
		{MerchantID: merchant.Id, Name: "冰豆浆", Description: "冷藏现磨豆浆", PriceCents: 600, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1544145945-f90425340c7e?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 2},
		{MerchantID: merchant.Id, Name: "酸梅汤", Description: "清爽解腻", PriceCents: 800, Stock: 9999, ImageURL: "https://images.unsplash.com/photo-1551024709-8f23befc6f87?auto=format&fit=crop&w=800&q=80", Status: bizmodel.ProductStatusOnSale, SortOrder: 1},
	}
	for _, item := range products {
		product := item
		if err := db.Where("merchant_id = ? AND name = ?", merchant.Id, item.Name).FirstOrCreate(&product, item).Error; err != nil {
			return err
		}
		if err := db.Model(&product).Updates(item).Error; err != nil {
			return err
		}
	}

	now := time.Now()
	session := bizmodel.StallSession{
		MerchantID:       merchant.Id,
		Status:           bizmodel.StatusActive,
		Lat:              22.3193,
		Lng:              114.1694,
		Address:          "旺角地铁站 A1 出口",
		PhotoURL:         "https://images.unsplash.com/photo-1514933651103-005eec06c04b?auto=format&fit=crop&w=1200&q=80",
		LocationAccuracy: 25,
		StartedAt:        now.Add(-30 * time.Minute),
		ExpectedEndAt:    now.Add(6 * time.Hour),
	}
	if err := db.Where("merchant_id = ? AND status = ?", merchant.Id, bizmodel.StatusActive).FirstOrCreate(&session, session).Error; err != nil {
		return err
	}
	session.Lat = 22.3193
	session.Lng = 114.1694
	session.Address = "旺角地铁站 A1 出口"
	session.PhotoURL = "https://images.unsplash.com/photo-1514933651103-005eec06c04b?auto=format&fit=crop&w=1200&q=80"
	session.LocationAccuracy = 25
	session.StartedAt = now.Add(-30 * time.Minute)
	session.ExpectedEndAt = now.Add(6 * time.Hour)
	session.EndedAt = nil
	if err := db.Select("merchant_id", "lat", "lng", "address", "photo_url", "location_accuracy", "started_at", "expected_end_at", "ended_at").Save(&session).Error; err != nil {
		return err
	}

	return seedNearbyDemoMerchants(db)
}

type demoMerchantSeed struct {
	Code         string
	Name         string
	Category     string
	Area         string
	Lat          float64
	Lng          float64
	AvatarURL    string
	PhotoURL     string
	Announcement string
	Products     []demoProductSeed
}

type demoProductSeed struct {
	Name        string
	Description string
	PriceCents  int64
	ImageURL    string
}

func seedNearbyDemoMerchants(db *gorm.DB) error {
	seeds := []demoMerchantSeed{
		{Code: "coffee-east", Name: "青榕手冲咖啡车", Category: "咖啡饮品", Area: "创意园东门", Lat: 22.32015, Lng: 114.17036, AvatarURL: "https://images.unsplash.com/photo-1495474472287-4d71bcdd2085?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1517701604599-bb29b565090c?auto=format&fit=crop&w=1200&q=80", Announcement: "手冲、美式、拿铁，午后常驻园区东门。", Products: []demoProductSeed{{"冰美式", "清爽低糖", 1800, "https://images.unsplash.com/photo-1461023058943-07fcbe16d735?auto=format&fit=crop&w=800&q=80"}, {"拿铁", "热/冰可选", 2200, "https://images.unsplash.com/photo-1517701604599-bb29b565090c?auto=format&fit=crop&w=800&q=80"}, {"燕麦拿铁", "燕麦奶版本", 2600, "https://images.unsplash.com/photo-1509042239860-f550ce710b93?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "fruit-south", Name: "甜橙鲜切水果摊", Category: "水果鲜切", Area: "社区南门", Lat: 22.31872, Lng: 114.16875, AvatarURL: "https://images.unsplash.com/photo-1619566636858-adf3ef46400b?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1490474418585-ba9bad8fd0ea?auto=format&fit=crop&w=1200&q=80", Announcement: "现切水果盒，支持少糖酸奶杯。", Products: []demoProductSeed{{"综合水果盒", "当日鲜切", 1600, "https://images.unsplash.com/photo-1490474418585-ba9bad8fd0ea?auto=format&fit=crop&w=800&q=80"}, {"芒果酸奶杯", "芒果加酸奶", 1800, "https://images.unsplash.com/photo-1488477181946-6428a0291777?auto=format&fit=crop&w=800&q=80"}, {"西瓜杯", "冰镇西瓜", 1200, "https://images.unsplash.com/photo-1563114773-84221bd62daa?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "bbq-night", Name: "老周夜宵烧烤", Category: "夜宵烧烤", Area: "青榕广场", Lat: 22.31985, Lng: 114.16791, AvatarURL: "https://images.unsplash.com/photo-1529692236671-f1f6cf9683ba?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1555939594-58d7cb561ad1?auto=format&fit=crop&w=1200&q=80", Announcement: "傍晚出摊，鸡翅、肉串、烤面筋。", Products: []demoProductSeed{{"羊肉串", "五串起", 1500, "https://images.unsplash.com/photo-1555939594-58d7cb561ad1?auto=format&fit=crop&w=800&q=80"}, {"烤鸡翅", "焦香微辣", 1200, "https://images.unsplash.com/photo-1529692236671-f1f6cf9683ba?auto=format&fit=crop&w=800&q=80"}, {"烤面筋", "秘制酱料", 800, "https://images.unsplash.com/photo-1544025162-d76694265947?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "lunch-box", Name: "阿兰便当快餐", Category: "便当快餐", Area: "福民大厦西侧", Lat: 22.32048, Lng: 114.16854, AvatarURL: "https://images.unsplash.com/photo-1546069901-ba9599a7e63c?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1547592180-85f173990554?auto=format&fit=crop&w=1200&q=80", Announcement: "工作日午餐便当，荤素搭配。", Products: []demoProductSeed{{"鸡腿饭", "卤鸡腿套餐", 2600, "https://images.unsplash.com/photo-1546069901-ba9599a7e63c?auto=format&fit=crop&w=800&q=80"}, {"番茄牛腩饭", "招牌热卖", 3200, "https://images.unsplash.com/photo-1512058564366-18510be2db19?auto=format&fit=crop&w=800&q=80"}, {"素菜双拼饭", "清爽少油", 2200, "https://images.unsplash.com/photo-1512621776951-a57141f2eefd?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "dumpling", Name: "胖姐手工水饺", Category: "早餐小吃", Area: "旺角街市口", Lat: 22.31895, Lng: 114.17052, AvatarURL: "https://images.unsplash.com/photo-1496116218417-1a781b1c416c?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1496116218417-1a781b1c416c?auto=format&fit=crop&w=1200&q=80", Announcement: "现包水饺、煎饺，早晚都在。", Products: []demoProductSeed{{"猪肉白菜水饺", "一盒十二只", 1800, "https://images.unsplash.com/photo-1496116218417-1a781b1c416c?auto=format&fit=crop&w=800&q=80"}, {"韭菜鸡蛋煎饺", "外脆里嫩", 1600, "https://images.unsplash.com/photo-1601050690597-df0568f70950?auto=format&fit=crop&w=800&q=80"}, {"紫菜蛋花汤", "热汤一杯", 600, "https://images.unsplash.com/photo-1547592166-23ac45744acd?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "noodle-cart", Name: "陈记车仔面", Category: "便当快餐", Area: "新汉大楼下", Lat: 22.32103, Lng: 114.16986, AvatarURL: "https://images.unsplash.com/photo-1569718212165-3a8278d5f624?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1569718212165-3a8278d5f624?auto=format&fit=crop&w=1200&q=80", Announcement: "自选配料，鱼蛋牛丸萝卜。", Products: []demoProductSeed{{"招牌车仔面", "三拼配料", 2800, "https://images.unsplash.com/photo-1569718212165-3a8278d5f624?auto=format&fit=crop&w=800&q=80"}, {"咖喱鱼蛋", "一份八粒", 1200, "https://images.unsplash.com/photo-1519708227418-c8fd9a32b7a2?auto=format&fit=crop&w=800&q=80"}, {"牛丸汤", "弹牙牛丸", 1400, "https://images.unsplash.com/photo-1547592166-23ac45744acd?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "tea-cart", Name: "小满柠檬茶", Category: "咖啡饮品", Area: "旺角消防局旁", Lat: 22.31957, Lng: 114.17114, AvatarURL: "https://images.unsplash.com/photo-1556679343-c7306c1976bc?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1556679343-c7306c1976bc?auto=format&fit=crop&w=1200&q=80", Announcement: "手打柠檬茶，下午茶常驻。", Products: []demoProductSeed{{"鸭屎香柠檬茶", "招牌手打", 1800, "https://images.unsplash.com/photo-1556679343-c7306c1976bc?auto=format&fit=crop&w=800&q=80"}, {"冻柠茶", "经典港式", 1400, "https://images.unsplash.com/photo-1497534446932-c925b458314e?auto=format&fit=crop&w=800&q=80"}, {"百香果茶", "酸甜清爽", 1600, "https://images.unsplash.com/photo-1551024709-8f23befc6f87?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "rice-roll", Name: "银记肠粉车", Category: "早餐小吃", Area: "福祥大厦门口", Lat: 22.32082, Lng: 114.16762, AvatarURL: "https://images.unsplash.com/photo-1627308595229-7830a5c91f9f?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1604908176997-125f25cc6f3d?auto=format&fit=crop&w=1200&q=80", Announcement: "现蒸肠粉，早餐和宵夜都有。", Products: []demoProductSeed{{"鲜虾肠粉", "鲜虾仁", 1800, "https://images.unsplash.com/photo-1604908176997-125f25cc6f3d?auto=format&fit=crop&w=800&q=80"}, {"牛肉肠粉", "滑嫩牛肉", 1700, "https://images.unsplash.com/photo-1562967916-eb82221dfb36?auto=format&fit=crop&w=800&q=80"}, {"鸡蛋肠粉", "经典早餐", 1200, "https://images.unsplash.com/photo-1482049016688-2d3e1b311543?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "burger-mini", Name: "迷你汉堡研究所", Category: "便当快餐", Area: "宏达金属建材门前", Lat: 22.31831, Lng: 114.16995, AvatarURL: "https://images.unsplash.com/photo-1550547660-d9450f859349?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1550547660-d9450f859349?auto=format&fit=crop&w=1200&q=80", Announcement: "小份汉堡，适合下午加餐。", Products: []demoProductSeed{{"牛肉芝士堡", "迷你双层", 2400, "https://images.unsplash.com/photo-1550547660-d9450f859349?auto=format&fit=crop&w=800&q=80"}, {"鸡排堡", "香脆鸡排", 2200, "https://images.unsplash.com/photo-1562967916-eb82221dfb36?auto=format&fit=crop&w=800&q=80"}, {"薯条杯", "现炸薯条", 1200, "https://images.unsplash.com/photo-1573080496219-bb080dd4f877?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "soup-warm", Name: "暖胃粥铺", Category: "早餐小吃", Area: "长发工业大厦", Lat: 22.32134, Lng: 114.16892, AvatarURL: "https://images.unsplash.com/photo-1517673132405-a56a62b18caf?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1517673132405-a56a62b18caf?auto=format&fit=crop&w=1200&q=80", Announcement: "早晨热粥，晚间小碗汤。", Products: []demoProductSeed{{"皮蛋瘦肉粥", "经典热粥", 1500, "https://images.unsplash.com/photo-1517673132405-a56a62b18caf?auto=format&fit=crop&w=800&q=80"}, {"南瓜小米粥", "清甜软糯", 1200, "https://images.unsplash.com/photo-1609501676725-7186f017a4b7?auto=format&fit=crop&w=800&q=80"}, {"葱油饼", "搭配热粥", 800, "https://images.unsplash.com/photo-1504674900247-0877df9cc836?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "fried-chicken", Name: "咔滋炸鸡摊", Category: "夜宵烧烤", Area: "客家小城旁", Lat: 22.31858, Lng: 114.16732, AvatarURL: "https://images.unsplash.com/photo-1562967916-eb82221dfb36?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1562967916-eb82221dfb36?auto=format&fit=crop&w=1200&q=80", Announcement: "现炸鸡块，夜宵热门。", Products: []demoProductSeed{{"盐酥鸡", "招牌一份", 1800, "https://images.unsplash.com/photo-1562967916-eb82221dfb36?auto=format&fit=crop&w=800&q=80"}, {"香辣鸡翅", "两只装", 1600, "https://images.unsplash.com/photo-1529692236671-f1f6cf9683ba?auto=format&fit=crop&w=800&q=80"}, {"甘梅地瓜", "甜口小吃", 1200, "https://images.unsplash.com/photo-1573080496219-bb080dd4f877?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "taco-truck", Name: "街角卷饼车", Category: "便当快餐", Area: "百利达广场", Lat: 22.32018, Lng: 114.16686, AvatarURL: "https://images.unsplash.com/photo-1565299507177-b0ac66763828?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1565299507177-b0ac66763828?auto=format&fit=crop&w=1200&q=80", Announcement: "墨西哥卷饼，午餐供应。", Products: []demoProductSeed{{"牛肉卷饼", "酸奶酱", 2600, "https://images.unsplash.com/photo-1565299507177-b0ac66763828?auto=format&fit=crop&w=800&q=80"}, {"鸡肉卷饼", "香辣鸡肉", 2400, "https://images.unsplash.com/photo-1552332386-f8dd00dc2f85?auto=format&fit=crop&w=800&q=80"}, {"玉米片", "配莎莎酱", 1400, "https://images.unsplash.com/photo-1513456852971-30c0b8199d4d?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "dessert-mango", Name: "芒果甜品站", Category: "水果鲜切", Area: "华富冰室门口", Lat: 22.31793, Lng: 114.16821, AvatarURL: "https://images.unsplash.com/photo-1488477181946-6428a0291777?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1488477181946-6428a0291777?auto=format&fit=crop&w=1200&q=80", Announcement: "芒果、椰奶、西米露。", Products: []demoProductSeed{{"杨枝甘露", "招牌甜品", 2200, "https://images.unsplash.com/photo-1488477181946-6428a0291777?auto=format&fit=crop&w=800&q=80"}, {"芒果班戟", "两枚装", 1800, "https://images.unsplash.com/photo-1565958011703-44f9829ba187?auto=format&fit=crop&w=800&q=80"}, {"椰汁西米露", "冰爽甜品", 1600, "https://images.unsplash.com/photo-1499636136210-6f4ee915583e?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "corn-roast", Name: "阿勇烤玉米", Category: "夜宵烧烤", Area: "利友街口", Lat: 22.32161, Lng: 114.17078, AvatarURL: "https://images.unsplash.com/photo-1551754655-cd27e38d2076?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1551754655-cd27e38d2076?auto=format&fit=crop&w=1200&q=80", Announcement: "炭烤玉米和烤红薯。", Products: []demoProductSeed{{"炭烤玉米", "甜玉米", 1000, "https://images.unsplash.com/photo-1551754655-cd27e38d2076?auto=format&fit=crop&w=800&q=80"}, {"烤红薯", "软糯香甜", 900, "https://images.unsplash.com/photo-1518977676601-b53f82aba655?auto=format&fit=crop&w=800&q=80"}, {"烤土豆片", "椒盐味", 800, "https://images.unsplash.com/photo-1505576399279-565b52d4ac71?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "juice-bar", Name: "活力鲜榨果汁", Category: "水果鲜切", Area: "明轩玻璃店旁", Lat: 22.31754, Lng: 114.17016, AvatarURL: "https://images.unsplash.com/photo-1622597467836-f3285f2131b8?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1622597467836-f3285f2131b8?auto=format&fit=crop&w=1200&q=80", Announcement: "鲜榨果汁，不加糖可选。", Products: []demoProductSeed{{"橙汁", "鲜榨一杯", 1600, "https://images.unsplash.com/photo-1621506289937-a8e4df240d0b?auto=format&fit=crop&w=800&q=80"}, {"苹果胡萝卜汁", "轻食搭配", 1800, "https://images.unsplash.com/photo-1622597467836-f3285f2131b8?auto=format&fit=crop&w=800&q=80"}, {"牛油果奶昔", "浓郁顺滑", 2200, "https://images.unsplash.com/photo-1553530666-ba11a7da3888?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "sushi-box", Name: "小林寿司盒", Category: "便当快餐", Area: "福星工厂大厦", Lat: 22.32188, Lng: 114.16931, AvatarURL: "https://images.unsplash.com/photo-1579871494447-9811cf80d66c?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1579871494447-9811cf80d66c?auto=format&fit=crop&w=1200&q=80", Announcement: "午间寿司盒，售完即止。", Products: []demoProductSeed{{"三文鱼寿司盒", "六枚装", 3200, "https://images.unsplash.com/photo-1579871494447-9811cf80d66c?auto=format&fit=crop&w=800&q=80"}, {"加州卷", "八枚装", 2800, "https://images.unsplash.com/photo-1553621042-f6e147245754?auto=format&fit=crop&w=800&q=80"}, {"味噌汤", "热汤搭配", 800, "https://images.unsplash.com/photo-1547592166-23ac45744acd?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "baozi", Name: "老面包子铺", Category: "早餐小吃", Area: "登发楼附近", Lat: 22.31782, Lng: 114.16674, AvatarURL: "https://images.unsplash.com/photo-1601050690597-df0568f70950?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1601050690597-df0568f70950?auto=format&fit=crop&w=1200&q=80", Announcement: "老面发酵，早高峰供应。", Products: []demoProductSeed{{"鲜肉包", "一笼四个", 1200, "https://images.unsplash.com/photo-1601050690597-df0568f70950?auto=format&fit=crop&w=800&q=80"}, {"香菇菜包", "素馅", 1000, "https://images.unsplash.com/photo-1496116218417-1a781b1c416c?auto=format&fit=crop&w=800&q=80"}, {"豆浆包子套餐", "豆浆加两包", 1300, "https://images.unsplash.com/photo-1544145945-f90425340c7e?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "ice-powder", Name: "冰粉凉虾小摊", Category: "水果鲜切", Area: "荣富大厦", Lat: 22.32074, Lng: 114.17156, AvatarURL: "https://images.unsplash.com/photo-1499636136210-6f4ee915583e?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1499636136210-6f4ee915583e?auto=format&fit=crop&w=1200&q=80", Announcement: "夏日冰粉，红糖糍粑。", Products: []demoProductSeed{{"红糖冰粉", "经典红糖", 1200, "https://images.unsplash.com/photo-1499636136210-6f4ee915583e?auto=format&fit=crop&w=800&q=80"}, {"水果冰粉", "鲜果加料", 1600, "https://images.unsplash.com/photo-1490474418585-ba9bad8fd0ea?auto=format&fit=crop&w=800&q=80"}, {"红糖糍粑", "现炸小份", 1400, "https://images.unsplash.com/photo-1551024506-0bccd828d307?auto=format&fit=crop&w=800&q=80"}}},
		{Code: "skewer-tofu", Name: "豆腐串串香", Category: "夜宵烧烤", Area: "云峰楼下", Lat: 22.31912, Lng: 114.16621, AvatarURL: "https://images.unsplash.com/photo-1544025162-d76694265947?auto=format&fit=crop&w=800&q=80", PhotoURL: "https://images.unsplash.com/photo-1544025162-d76694265947?auto=format&fit=crop&w=1200&q=80", Announcement: "串串、豆腐、蔬菜，辣度可调。", Products: []demoProductSeed{{"豆腐串", "五串", 1000, "https://images.unsplash.com/photo-1544025162-d76694265947?auto=format&fit=crop&w=800&q=80"}, {"蔬菜串", "当日蔬菜", 900, "https://images.unsplash.com/photo-1546069901-ba9599a7e63c?auto=format&fit=crop&w=800&q=80"}, {"牛肉串", "三串", 1500, "https://images.unsplash.com/photo-1555939594-58d7cb561ad1?auto=format&fit=crop&w=800&q=80"}}},
	}
	now := time.Now()
	for index, seed := range seeds {
		phone := fmt.Sprintf("13800138%03d", index+2)
		user := bizmodel.User{}
		if err := db.Where("phone = ?", phone).FirstOrCreate(&user, user).Error; err != nil {
			return err
		}
		user.Phone = phone
		user.Username = phone
		user.Nickname = seed.Name
		user.Password = user.SetPassword("123456")
		user.IsValid = true
		user.IsLock = false
		if err := db.Select("phone", "username", "nickname", "password", "is_valid", "is_lock").Save(&user).Error; err != nil {
			return err
		}
		merchant := bizmodel.Merchant{MerchantItem: bizmodel.MerchantItem{UserID: user.Id, DisplayName: seed.Name}}
		if err := db.Where("user_id = ?", user.Id).FirstOrCreate(&merchant, merchant).Error; err != nil {
			return err
		}
		merchant.UserID = user.Id
		merchant.DisplayName = seed.Name
		merchant.Phone = phone
		merchant.Category = seed.Category
		merchant.AvatarURL = seed.AvatarURL
		merchant.Announcement = seed.Announcement
		merchant.ContactPhone = phone
		merchant.Status = bizmodel.StatusActive
		merchant.VerifyStatus = bizmodel.VerifyVerified
		merchant.DisabledReason = ""
		merchant.ShareCode = fmt.Sprintf("D%05d", index+2)
		if err := db.Select("user_id", "display_name", "phone", "category", "avatar_url", "announcement", "contact_phone", "status", "verify_status", "disabled_reason", "share_code").Save(&merchant).Error; err != nil {
			return err
		}
		if err := db.Model(&user).Update("merchant_id", merchant.Id).Error; err != nil {
			return err
		}
		for sort, productSeed := range seed.Products {
			product := bizmodel.ProductItem{MerchantID: merchant.Id, Name: productSeed.Name}
			item := bizmodel.ProductItem{MerchantID: merchant.Id, Name: productSeed.Name, Description: productSeed.Description, PriceCents: productSeed.PriceCents, Stock: 9999, ImageURL: productSeed.ImageURL, Status: bizmodel.ProductStatusOnSale, SortOrder: len(seed.Products) - sort}
			if err := db.Where("merchant_id = ? AND name = ?", merchant.Id, productSeed.Name).FirstOrCreate(&product, item).Error; err != nil {
				return err
			}
			if err := db.Model(&product).Updates(item).Error; err != nil {
				return err
			}
		}
		session := bizmodel.StallSession{MerchantID: merchant.Id, Status: bizmodel.StatusActive}
		if err := db.Where("merchant_id = ? AND status = ?", merchant.Id, bizmodel.StatusActive).FirstOrCreate(&session, bizmodel.StallSession{
			MerchantID: merchant.Id, Status: bizmodel.StatusActive, Lat: seed.Lat, Lng: seed.Lng, Address: seed.Area, PhotoURL: seed.PhotoURL, LocationAccuracy: 20 + index%18, StartedAt: now.Add(-time.Duration(10+index*3) * time.Minute), ExpectedEndAt: now.Add(time.Duration(4+index%5) * time.Hour),
		}).Error; err != nil {
			return err
		}
		session.Lat = seed.Lat
		session.Lng = seed.Lng
		session.Address = seed.Area
		session.PhotoURL = seed.PhotoURL
		session.LocationAccuracy = 20 + index%18
		session.StartedAt = now.Add(-time.Duration(10+index*3) * time.Minute)
		session.ExpectedEndAt = now.Add(time.Duration(4+index%5) * time.Hour)
		session.EndedAt = nil
		if err := db.Select("merchant_id", "lat", "lng", "address", "photo_url", "location_accuracy", "started_at", "expected_end_at", "ended_at").Save(&session).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedSysAdmin(db *gorm.DB) error {
	role := sysmodel.Role{RoleBase: sysmodel.RoleBase{Tree: tree.Tree{Id: 1, Name: "super_admin"}, Status: 1, DefaultMenu: "/admin"}}
	if err := db.Where("id = ?", role.Id).FirstOrCreate(&role, role).Error; err != nil {
		return err
	}
	user := sysmodel.User{Info: sysmodel.User{}.Info}
	user.Id = 1
	user.Username = "admin"
	user.Nickname = "系统管理员"
	user.Phone = "13800000000"
	user.IsValid = true
	user.Password = user.SetPassword("admin123")
	var existing sysmodel.User
	err := db.Where("id = ?", user.Id).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&user).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&existing).Select("username", "nickname", "phone", "is_valid", "password").Updates(user).Error
}
