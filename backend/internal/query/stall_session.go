package query

import (
	"strings"
	"time"

	gkkhandler "gkk/handler"
	gkkmodel "gkk/model"

	bizmodel "github.com/gkk/stall-location/backend/internal/model"
	"gorm.io/gorm"
)

type StallSessionQuery struct {
	MerchantID uint   `query:"merchant_id" find:"="`
	Status     string `query:"status" find:"="`
	gkkmodel.Period[uint]
}

func (q StallSessionQuery) DBS(db *gorm.DB) *gorm.DB {
	return gkkhandler.Encode2DB(q.Period.DBS(db), q).Preload("Merchant")
}

func (q StallSessionQuery) DB(db *gorm.DB) *gorm.DB {
	return db.Preload("Merchant")
}

type PublicActiveStallSessionQuery struct {
	gkkmodel.IdRequired[uint]
}

func (q PublicActiveStallSessionQuery) DB(db *gorm.DB) *gorm.DB {
	if q.Id == 0 {
		return db.Where("1 = 0")
	}
	db = db.
		Joins("JOIN merchants ON merchants.id = stall_sessions.merchant_id").
		Where("stall_sessions.status = ? AND stall_sessions.expected_end_at > ?", bizmodel.StatusActive, time.Now()).
		Where("merchants.status = ? AND merchants.verify_status = ?", bizmodel.StatusActive, bizmodel.VerifyVerified).
		Preload("Merchant")
	return db.Where("merchants.id = ?", q.Id)
}

type NearbyStallQuery struct {
	Lat        *float64  `query:"lat" find:"-"`
	Lng        *float64  `query:"lng" find:"-"`
	Limit      int       `query:"limit" find:"-"`
	Q          string    `query:"q" find:"-"`
	Category   string    `query:"category" find:"-"`
	Categories []string  `query:"categories" find:"-"`
	MinLat     *float64  `query:"min_lat" find:"-"`
	MaxLat     *float64  `query:"max_lat" find:"-"`
	MinLng     *float64  `query:"min_lng" find:"-"`
	MaxLng     *float64  `query:"max_lng" find:"-"`
	Zoom       string    `query:"zoom" find:"-"`
	Now        time.Time `query:"-" json:"-" find:"-"`
	gkkmodel.PageSize
}

func (q NearbyStallQuery) DBS(db *gorm.DB) *gorm.DB {
	db = db.Joins("JOIN merchants ON merchants.id = stall_sessions.merchant_id").
		Where("merchants.status = ? AND merchants.verify_status = ?", bizmodel.StatusActive, bizmodel.VerifyVerified).
		Where("stall_sessions.status = ? ", bizmodel.StatusActive)
	if q.HasBounds() {
		minLat, maxLat, minLng, maxLng := q.Bounds()
		db = db.Where("stall_sessions.lat BETWEEN ? AND ? AND stall_sessions.lng BETWEEN ? AND ?", minLat, maxLat, minLng, maxLng)
	}
	if categories := q.NormalizedCategories(); len(categories) > 0 {
		db = db.Where("merchants.category IN ?", categories)
	}
	if search := q.SearchText(); search != "" {
		like := "%" + search + "%"
		db = db.Where("(merchants.display_name LIKE ? OR merchants.category LIKE ? OR merchants.announcement LIKE ? OR stall_sessions.address LIKE ?)", like, like, like, like)
	}
	return db.Preload("Merchant")
}

func (q NearbyStallQuery) Order() string {
	return "stall_sessions.started_at DESC, stall_sessions.id DESC"
}

func (q NearbyStallQuery) PS() *gkkmodel.PageSize {
	ps := q.PageSize
	ps.Size = q.ResultLimit()
	return &ps
}

func (q NearbyStallQuery) HasLocation() bool {
	return q.Lat != nil && q.Lng != nil
}

func (q NearbyStallQuery) Location() (float64, float64) {
	return deref(q.Lat), deref(q.Lng)
}

func (q NearbyStallQuery) HasBounds() bool {
	return q.MinLat != nil && q.MaxLat != nil && q.MinLng != nil && q.MaxLng != nil
}

func (q NearbyStallQuery) Bounds() (float64, float64, float64, float64) {
	return deref(q.MinLat), deref(q.MaxLat), deref(q.MinLng), deref(q.MaxLng)
}

func (q NearbyStallQuery) ResultLimit() int {
	limit := q.Limit
	if limit <= 0 {
		limit = q.Size
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 300 {
		limit = 300
	}
	return limit
}

func (q NearbyStallQuery) SearchText() string {
	return strings.TrimSpace(q.Q)
}

func (q NearbyStallQuery) NormalizedCategories() []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(q.Categories)+1)
	appendValue := func(value string) {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" && !seen[part] {
				seen[part] = true
				out = append(out, part)
			}
		}
	}
	appendValue(q.Category)
	for _, category := range q.Categories {
		appendValue(category)
	}
	return out
}

func (q NearbyStallQuery) now() time.Time {
	if !q.Now.IsZero() {
		return q.Now
	}
	return time.Now()
}

func deref(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

type ActiveStallSessionQuery struct {
	MerchantID uint   `query:"merchant_id" find:"="`
	Category   string `query:"category" find:"-"`
	Q          string `query:"q" find:"-"`
	gkkmodel.Period[uint]
}

func (q ActiveStallSessionQuery) DBS(db *gorm.DB) *gorm.DB {
	db = q.Period.DBS(db).
		Joins("JOIN merchants ON merchants.id = stall_sessions.merchant_id").
		Where("stall_sessions.status = ? AND stall_sessions.expected_end_at > ?", bizmodel.StatusActive, time.Now()).
		Where("merchants.status = ? AND merchants.verify_status = ?", bizmodel.StatusActive, bizmodel.VerifyVerified)
	if q.MerchantID > 0 {
		db = db.Where("stall_sessions.merchant_id = ?", q.MerchantID)
	}
	if category := strings.TrimSpace(q.Category); category != "" {
		db = db.Where("merchants.category = ?", category)
	}
	if search := strings.TrimSpace(q.Q); search != "" {
		like := "%" + search + "%"
		db = db.Where("merchants.display_name LIKE ? OR stall_sessions.address LIKE ?", like, like)
	}
	return gkkhandler.Encode2DB(db, q).Preload("Merchant")
}

func (q ActiveStallSessionQuery) Order() string {
	return "stall_sessions.started_at desc,stall_sessions.id desc"
}
