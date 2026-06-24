package service

import (
	"math"
	"sort"
	"strings"
	"time"

	"gkk/expect"
	"gkk/model"
	"gkk/orm"

	bizquery "github.com/gkk/stall-location/backend/internal/query"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type StallService struct {
	DB  *gorm.DB
	Now clock
}

type StallMapItem struct {
	Merchant         *PublicMerchant    `json:"merchant,omitempty"`
	StallSession     PublicStallSession `json:"stall_session"`
	DistanceMeters   *int               `json:"distance_meters,omitempty"`
	WalkMinutes      *int               `json:"walk_minutes,omitempty"`
	DisplayStatus    string             `json:"display_status"`
	EntryMode        string             `json:"entry_mode"`
	LocationAccuracy int                `json:"location_accuracy"`
	LastOnlineAt     time.Time          `json:"last_online_at"`
}

type StartStallRequest struct {
	Lat              float64    `json:"lat" validate:"required"`
	Lng              float64    `json:"lng" validate:"required"`
	Address          string     `json:"address" validate:"required"`
	PhotoURL         string     `json:"photo_url"`
	LocationAccuracy int        `json:"location_accuracy"`
	ExpectedEndAt    *time.Time `json:"expected_end_at"`
}

func (s *StallService) NearbyAction(c fiber.Ctx, req bizquery.NearbyStallQuery) (any, error) {
	req.Now = serviceNow(s.Now)
	tx := req.DBS(s.DB.Model(&StallSession{})).Order(req.Order())
	sessionRows, total := orm.List[StallSession](tx, req.PS())
	sessions := stallSessionValues(sessionRows)
	lat, lng := req.Location()
	items := make([]StallMapItem, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, s.stallItem(session, "nearby", lat, lng, req.HasLocation()))
	}
	if req.HasLocation() {
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].DistanceMeters == nil || items[j].DistanceMeters == nil {
				return items[i].DistanceMeters != nil
			}
			return *items[i].DistanceMeters < *items[j].DistanceMeters
		})
	}
	return model.List[StallMapItem]{Data: items, Total: total}, nil
}

func (s *StallService) ExpireSessions() {
	now := time.Now()
	s.DB.Model(&StallSession{}).
		Where("status = ? AND expected_end_at <= ?", StatusActive, now).
		Select("status", "ended_at").Updates(StallSession{Status: StatusEnded, EndedAt: &now})
}

func stallSessionValues(rows []*StallSession) []StallSession {
	sessions := make([]StallSession, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			sessions = append(sessions, *row)
		}
	}
	return sessions
}

func (s *StallService) PublicProductsAction(_ fiber.Ctx, req bizquery.PublicProductQuery) (model.List[PublicProduct], error) {
	tx := req.DBS(s.DB.Model(&Product{})).Order(req.Order())
	rows, total := orm.List[Product](tx, req.PS())
	products := make([]PublicProduct, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			products = append(products, row.Public())
		}
	}
	return model.List[PublicProduct]{Data: products, Total: total}, nil
}

func (s *StallService) Start(c fiber.Ctx, req StartStallRequest) (any, error) {
	merchantID := *User(c).MerchantID
	now := serviceNow(s.Now)
	expectedEndAt := now.Add(6 * time.Hour)
	if req.ExpectedEndAt != nil {
		expectedEndAt = *req.ExpectedEndAt
	}
	if !expectedEndAt.After(now) {
		return nil, validation("expected_end_at 必须晚于当前时间")
	}
	return nil, withTx(s.DB, func(tx *gorm.DB) error {
		end := now
		if db := tx.Model(&StallSession{}).Where("merchant_id = ? AND status = ?", merchantID, StatusActive).Select("status", "ended_at").Updates(StallSession{Status: StatusEnded, EndedAt: &end}); db.Error != nil {
			return expect.NDM(db, "关闭旧出摊失败")
		}
		session := StallSession{MerchantID: merchantID, Status: StatusActive, Lat: req.Lat, Lng: req.Lng, Address: strings.TrimSpace(req.Address), PhotoURL: strings.TrimSpace(req.PhotoURL), LocationAccuracy: req.LocationAccuracy, StartedAt: now, ExpectedEndAt: expectedEndAt}
		if db := tx.Create(&session); db.Error != nil {
			return expect.NDM(db, "开始出摊失败")
		}
		return nil
	})
}

func (s *StallService) End(c fiber.Ctx) (any, error) {
	merchantID := *User(c).MerchantID
	now := serviceNow(s.Now)
	return nil, withTx(s.DB, func(tx *gorm.DB) error {
		session := orm.First[StallSession](tx.Where("merchant_id = ? AND status = ?", merchantID, StatusActive).Order("started_at DESC"))
		if session.Id == 0 {
			return notFound("当前无有效出摊")
		}
		if db := tx.Model(&StallSession{}).
			Where("id = ? AND merchant_id = ? AND status = ?", session.Id, merchantID, StatusActive).
			Select("status", "ended_at").
			Updates(StallSession{Status: StatusEnded, EndedAt: &now}); db.Error != nil {
			return expect.NDM(db, "结束出摊失败")
		}
		return closePendingOrdersForMerchant(tx, merchantID)
	})
}

func closePendingOrdersForMerchant(tx *gorm.DB, merchantID uint) error {
	orders := orm.Find[Preorder](tx.Where("merchant_id = ? AND status = ?", merchantID, OrderPendingAccept))
	if len(orders) == 0 {
		return nil
	}
	orderIDs := make([]uint, 0, len(orders))
	for _, order := range orders {
		if err := restockClosedOrder(tx, &order); err != nil {
			return err
		}
		orderIDs = append(orderIDs, order.Id)
	}
	db := tx.Model(&Preorder{}).
		Where("merchant_id = ? AND status = ? AND id IN ?", merchantID, OrderPendingAccept, orderIDs).
		Update("status", OrderExpired)
	return expect.NDM(db, "关闭未接单订单失败")
}

func restockClosedOrder(tx *gorm.DB, order *Preorder) error {
	items := orm.Find[PreorderItem](tx.Where("order_id = ?", order.Id))
	for _, item := range items {
		if db := tx.Model(&Product{}).Where("id = ? AND merchant_id = ?", item.ProductID, order.MerchantID).UpdateColumn("stock", gorm.Expr("stock + ?", item.Quantity)); db.Error != nil {
			return expect.NDM(db, "回补库存失败")
		}
	}
	return nil
}

func (s *StallService) stallItem(session StallSession, mode string, userLat, userLng float64, hasUserLocation bool) StallMapItem {
	active := session.Status == StatusActive && session.ExpectedEndAt.After(serviceNow(s.Now))
	displayStatus := "unavailable"
	if active {
		displayStatus = "active"
	}
	item := StallMapItem{Merchant: publicMerchant(session.Merchant), StallSession: session.Public(), DisplayStatus: displayStatus, EntryMode: mode, LocationAccuracy: session.LocationAccuracy, LastOnlineAt: sessionLastOnlineAt(session)}
	if hasUserLocation {
		distance := int(math.Round(haversineMeters(userLat, userLng, session.Lat, session.Lng)))
		walk := int(math.Max(1, math.Round(float64(distance)/80.0)))
		item.DistanceMeters = &distance
		item.WalkMinutes = &walk
	}
	return item
}

func publicMerchant(merchant *Merchant) *PublicMerchant {
	if merchant == nil {
		return nil
	}
	res := merchant.Public()
	return &res
}

func sessionLastOnlineAt(session StallSession) time.Time {
	if session.EndedAt != nil && !session.EndedAt.IsZero() {
		return *session.EndedAt
	}
	if !session.UpdatedAt.IsZero() {
		return session.UpdatedAt
	}
	return session.StartedAt
}
