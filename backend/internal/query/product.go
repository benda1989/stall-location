package query

import (
	"strings"

	gkkhandler "gkk/handler"
	gkkmodel "gkk/model"

	bizmodel "github.com/gkk/stall-location/backend/internal/model"
	"gorm.io/gorm"
)

type ProductQuery struct {
	MerchantID uint   `query:"merchant_id" find:"="`
	Name       string `query:"name" find:"like"`
	Status     string `query:"status" find:"="`
	MinPrice   int64  `query:"min_price_cents" find:"-"`
	MaxPrice   int64  `query:"max_price_cents" find:"-"`
	StockLess  int    `query:"stock_less" find:"-"`
	gkkmodel.Period[uint]
}

func (q ProductQuery) DBS(db *gorm.DB) *gorm.DB {
	db = gkkhandler.Encode2DB(q.Period.DBS(db), q)
	if q.MinPrice > 0 {
		db = db.Where("price_cents >= ?", q.MinPrice)
	}
	if q.MaxPrice > 0 {
		db = db.Where("price_cents <= ?", q.MaxPrice)
	}
	if q.StockLess > 0 {
		db = db.Where("stock < ?", q.StockLess)
	}
	return db.Preload("Merchant")
}
func (q ProductQuery) Order() string { return ProductDisplayOrder() }

func ProductDisplayOrder() string {
	return "products.pinned_at IS NULL, products.pinned_at DESC, products.sort_order DESC, products.id DESC"
}

type PublicProductQuery struct {
	gkkmodel.IdReq[uint]
	MerchantID uint   `query:"merchant_id" find:"-"`
	Name       string `query:"name" find:"-"`
	gkkmodel.PageSize
}

func (q PublicProductQuery) DBS(db *gorm.DB) *gorm.DB {
	merchantID := q.Id
	if merchantID == 0 {
		merchantID = q.MerchantID
	}
	if merchantID == 0 {
		return db.Where("1 = 0")
	}
	db = db.
		Joins("JOIN merchants ON merchants.id = products.merchant_id").
		Where("products.merchant_id = ?", merchantID).
		Where("products.status = ?", bizmodel.ProductStatusOnSale).
		Where("merchants.status = ? AND merchants.verify_status = ?", bizmodel.StatusActive, bizmodel.VerifyVerified)
	if name := strings.TrimSpace(q.Name); name != "" {
		like := "%" + name + "%"
		db = db.Where("products.name LIKE ? OR products.description LIKE ?", like, like)
	}
	return db.Preload("Merchant")
}

func (q PublicProductQuery) Order() string { return ProductDisplayOrder() }
