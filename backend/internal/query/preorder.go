package query

import (
	"strings"

	gkkhandler "gkk/handler"
	gkkmodel "gkk/model"

	"gorm.io/gorm"
)

type PreorderQuery struct {
	OrderNo        string `query:"order_no" find:"="`
	MerchantID     uint   `query:"merchant_id" find:"="`
	StallSessionID uint   `query:"stall_session_id" find:"="`
	UserID         uint   `query:"user_id" find:"="`
	CustomerPhone  string `query:"customer_phone" find:"="`
	Status         string `query:"status" find:"="`
	PaymentStatus  string `query:"payment_status" find:"="`
	gkkmodel.Period[uint]
}

func (q PreorderQuery) DBS(db *gorm.DB) *gorm.DB {
	return gkkhandler.Encode2DB(q.Period.DBS(db), q).Preload("Merchant").Preload("StallSession").Preload("Items")
}

type CustomerOrderQuery struct {
	OrderNo       string `query:"order_no" find:"="`
	CustomerPhone string `query:"customer_phone" find:"="`
	PaymentStatus string `query:"payment_status" find:"="`
	gkkmodel.Period[uint]
}

func (q CustomerOrderQuery) DBS(db *gorm.DB) *gorm.DB {
	return gkkhandler.Encode2DB(q.Period.DBS(db), q).Preload("Merchant").Preload("Items")
}

type CustomerOrderDetailQuery struct {
	gkkmodel.IdRequired[uint]
}

func (q CustomerOrderDetailQuery) DB(db *gorm.DB) *gorm.DB {
	return db.Where("id = ?", q.Id).Preload("Merchant").Preload("StallSession").Preload("Items")
}

type MerchantPreorderQuery struct {
	gkkmodel.IdReq[uint]
	gkkmodel.PageSize
	Status string `query:"status" find:"="`
}

func (q MerchantPreorderQuery) DBS(db *gorm.DB) *gorm.DB {
	q.Status = strings.TrimSpace(q.Status)
	return gkkhandler.Encode2DB(db, q).Preload("Merchant").Preload("StallSession").Preload("Items")
}
func (q MerchantPreorderQuery) Order() string { return "id DESC" }

type MerchantOrderQuery struct {
	PreorderQuery
	MerchantID uint `query:"merchant_id" find:"-"`
}

func (q MerchantOrderQuery) DBS(db *gorm.DB) *gorm.DB {
	return q.scopedDB(q.PreorderQuery.DBS(db))
}

func (q MerchantOrderQuery) scopedDB(db *gorm.DB) *gorm.DB {
	if q.MerchantID > 0 {
		db = db.Where("preorders.merchant_id = ?", q.MerchantID)
	}
	return db
}

type SysOrderQuery struct {
	PreorderQuery
	Category string `query:"category" find:"-"`
}

func (q SysOrderQuery) DBS(db *gorm.DB) *gorm.DB {
	return q.filterDB(gkkhandler.Encode2DBPrefix(q.Period.DBS(db), q.PreorderQuery, "preorders.").Preload("Merchant").Preload("StallSession").Preload("Items"))
}

func (q SysOrderQuery) filterDB(db *gorm.DB) *gorm.DB {
	category := strings.TrimSpace(q.Category)
	if category != "" {
		db = db.Joins("JOIN merchants ON merchants.id = preorders.merchant_id")
	}
	if category != "" {
		db = db.Where("merchants.category = ?", category)
	}
	return db
}
