package query

import (
	"strings"

	gkkhandler "gkk/handler"
	gkkmodel "gkk/model"

	bizmodel "github.com/gkk/stall-location/backend/internal/model"
	"gorm.io/gorm"
)

type MerchantQuery struct {
	UserID       uint   `query:"user_id" find:"="`
	Phone        string `query:"phone" find:"="`
	DisplayName  string `query:"display_name" find:"like"`
	Category     string `query:"category" find:"="`
	Status       string `query:"status" find:"="`
	VerifyStatus string `query:"verify_status" find:"="`
	gkkmodel.Period[uint]
}

func (q MerchantQuery) DBS(db *gorm.DB) *gorm.DB { return gkkhandler.Encode2DB(q.Period.DBS(db), q) }

type PublicMerchantQuery struct {
	Id        uint   `query:"id" json:"id,omitempty" find:"-" validate:"required_without=ShareCode"`
	ShareCode string `query:"share_code" json:"share_code,omitempty" find:"-" validate:"required_without=Id"`
}

func (q PublicMerchantQuery) DB(db *gorm.DB) *gorm.DB {
	db = db.Where("status = ? AND verify_status = ?", bizmodel.StatusActive, bizmodel.VerifyVerified)
	if q.Id != 0 {
		return db.Where("id = ?", q.Id)
	}
	shareCode := strings.TrimSpace(q.ShareCode)
	if shareCode == "" {
		return db.Where("1 = 0")
	}
	return db.Where("share_code = ?", shareCode)
}

type PublicMerchantSearchQuery struct {
	Category string `query:"category" find:"-"`
	Q        string `query:"q" find:"-"`
	gkkmodel.Period[uint]
}

func (q PublicMerchantSearchQuery) DBS(db *gorm.DB) *gorm.DB {
	return q.publicDB(q.Period.DBS(db))
}

func (q PublicMerchantSearchQuery) publicDB(db *gorm.DB) *gorm.DB {
	db = db.Where("status = ? AND verify_status = ?", bizmodel.StatusActive, bizmodel.VerifyVerified)
	if category := strings.TrimSpace(q.Category); category != "" {
		db = db.Where("category = ?", category)
	}
	if search := strings.TrimSpace(q.Q); search != "" {
		like := "%" + search + "%"
		db = db.Where("display_name LIKE ? OR category LIKE ? OR announcement LIKE ?", like, like, like)
	}
	return db
}
