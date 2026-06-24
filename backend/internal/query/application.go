package query

import (
	gkkhandler "gkk/handler"
	gkkmodel "gkk/model"

	"gorm.io/gorm"
)

type ApplicationQuery struct {
	ApplicationNo string `query:"application_no" find:"="`
	UserID        uint   `query:"user_id" find:"="`
	ContactPhone  string `query:"contact_phone" find:"="`
	MerchantName  string `query:"merchant_name" find:"like"`
	Category      string `query:"category" find:"="`
	Status        string `query:"status" find:"="`
	MerchantID    uint   `query:"merchant_id" find:"="`
	gkkmodel.Period[uint]
}

func (q ApplicationQuery) DBS(db *gorm.DB) *gorm.DB {
	return gkkhandler.Encode2DB(q.Period.DBS(db), q)
}

type ApplicationDetailQuery struct {
	Id uint `query:"id" find:"-" validate:"required"`
}

func (q ApplicationDetailQuery) DB(db *gorm.DB) *gorm.DB {
	if q.Id == 0 {
		return db.Where("1 = 0")
	}
	return db.Where("id = ?", q.Id).Preload("Merchant")
}
