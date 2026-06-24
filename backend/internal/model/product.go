package model

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"gkk/expect"
	gkkmodel "gkk/model"

	"github.com/gkk/stall-location/backend/internal/conf"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxMerchantPinnedProducts = 3

type Product struct {
	ProductItem
	Merchant *Merchant `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
}

// ProductItem is the minimal write model for merchant product CUD. Product is
// the full read model and embeds this item to keep one field contract.
type ProductItem struct {
	gkkmodel.IdReq[uint]
	gkkmodel.Timestamps
	MerchantID  uint       `json:"-" gorm:"index;not null"`
	Name        string     `json:"name" gorm:"size:120;index;not null" validate:"required"`
	Description string     `json:"description" gorm:"size:512"`
	PriceCents  int64      `json:"price_cents" gorm:"not null;default:0"`
	Stock       int        `json:"stock" gorm:"not null;default:0"`
	ImageURL    string     `json:"image_url" gorm:"size:512;not null" validate:"required"`
	Status      string     `json:"status" gorm:"size:24;index;default:on_sale"`
	PinnedAt    *time.Time `json:"pinned_at,omitempty" gorm:"index"`
	SortOrder   int        `json:"sort_order" gorm:"index;default:0"`
}

type ProductPinUpdate struct {
	gkkmodel.IdRequired[uint]
	PinnedAt *time.Time `json:"pinned_at,omitempty" gorm:"index"`
}
type ProductUnpinUpdate struct {
	ProductPinUpdate
}

func (ProductItem) TableName() string      { return "products" }
func (ProductPinUpdate) TableName() string { return "products" }

func (p ProductItem) Omit() string {
	return "id,merchant_id,pinned_at,created_at"
}

func (p ProductPinUpdate) Omit() string { return "id" }

func (p *ProductItem) BeforeSave(_ *gorm.DB) error {
	if p.Id == 0 {
		return nil
	}
	return p.validate()
}
func (p *ProductItem) BeforeCreate(tx *gorm.DB) error {
	p.PinnedAt = nil
	if err := p.validate(); err != nil {
		return err
	}
	return p.checkMerchantProductLimit(tx)
}

func (p *ProductItem) AfterCreate(tx *gorm.DB) error {
	return RefreshProductSummaries(tx, p.MerchantID)
}

func (p *ProductItem) AfterUpdate(tx *gorm.DB) error {
	if p.Status == ProductStatusOffSale && p.Id != 0 {
		db := tx.Model(&Product{}).Where("id = ?", p.Id).UpdateColumn("pinned_at", nil)
		if err := expect.NDM(db, "下架商品取消置顶失败"); err != nil {
			return err
		}
	}
	return refreshProductMerchantSummaries(tx, p.Id, p.MerchantID)
}

func (p *ProductItem) BeforeDelete(tx *gorm.DB) error {
	applyProductDeleteScope(p, tx)
	if p.MerchantID != 0 || p.Id == 0 {
		return nil
	}
	var product Product
	if err := tx.Select("id", "merchant_id").First(&product, "id = ?", p.Id).Error; err != nil {
		return nil
	}
	p.MerchantID = product.MerchantID
	return nil
}

func (p *ProductItem) AfterDelete(tx *gorm.DB) error {
	applyProductDeleteScope(p, tx)
	if p.MerchantID == 0 {
		return RefreshAllProductSummaries(tx)
	}
	return refreshProductSummaries(tx, p.MerchantID, p.Id)
}

func applyProductDeleteScope(p *ProductItem, tx *gorm.DB) {
	if p == nil || tx == nil || tx.Statement == nil {
		return
	}
	item, ok := tx.Statement.Clauses["WHERE"]
	if !ok {
		return
	}
	where, ok := item.Expression.(clause.Where)
	if !ok {
		return
	}
	for _, expr := range where.Exprs {
		switch value := expr.(type) {
		case clause.Eq:
			column := deleteScopeColumnName(value.Column)
			if strings.EqualFold(column, "id") {
				p.Id = uintFromAny(value.Value)
			}
			if strings.EqualFold(column, "merchant_id") {
				p.MerchantID = uintFromAny(value.Value)
			}
		case clause.Expr:
			sql := strings.ToLower(value.SQL)
			idIndex := strings.Index(sql, "id")
			merchantIndex := strings.Index(sql, "merchant_id")
			if idIndex >= 0 && merchantIndex < 0 && len(value.Vars) > 0 {
				p.Id = uintFromAny(value.Vars[0])
			}
			if idIndex >= 0 && merchantIndex >= 0 && idIndex < merchantIndex && len(value.Vars) > 1 {
				p.Id = uintFromAny(value.Vars[0])
				p.MerchantID = uintFromAny(value.Vars[1])
			} else if merchantIndex >= 0 && len(value.Vars) > 0 {
				p.MerchantID = uintFromAny(value.Vars[0])
			}
		}
	}
}

func deleteScopeColumnName(column any) string {
	switch value := column.(type) {
	case string:
		return strings.TrimPrefix(value, "products.")
	case clause.Column:
		return strings.TrimPrefix(value.Name, "products.")
	default:
		return ""
	}
}

func uintFromAny(value any) uint {
	switch v := value.(type) {
	case uint:
		return v
	case uint8:
		return uint(v)
	case uint16:
		return uint(v)
	case uint32:
		return uint(v)
	case uint64:
		return uint(v)
	case int:
		return uint(v)
	case int8:
		return uint(v)
	case int16:
		return uint(v)
	case int32:
		return uint(v)
	case int64:
		return uint(v)
	default:
		return 0
	}
}

func (p *ProductPinUpdate) BeforeSave(_ *gorm.DB) error {
	now := time.Now()
	p.PinnedAt = &now
	return nil
}

func (p *ProductPinUpdate) AfterUpdate(tx *gorm.DB) error {
	merchantID, err := onSaleProductMerchantID(tx, p.Id)
	if err != nil || merchantID == 0 {
		return err
	}
	if err := TrimMerchantPinnedProducts(tx, merchantID); err != nil {
		return err
	}
	return RefreshProductSummaries(tx, merchantID)
}

func (p *ProductUnpinUpdate) BeforeSave(_ *gorm.DB) error {
	p.PinnedAt = nil
	return nil
}

func (p *ProductUnpinUpdate) AfterUpdate(tx *gorm.DB) error {
	return refreshProductMerchantSummaries(tx, p.Id, 0)
}

func (p *ProductItem) checkMerchantProductLimit(tx *gorm.DB) error {
	limit := conf.MaxMerchantProducts()
	if p.MerchantID == 0 || limit <= 0 {
		return nil
	}
	var total int64
	if err := tx.Model(&Product{}).Where("merchant_id = ?", p.MerchantID).Count(&total).Error; err != nil {
		return expect.Wrap(err, http.StatusInternalServerError, expect.CodeCommonInternalError, "查询商户商品数量失败")
	}
	if total >= int64(limit) {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "商户商品数量已达上限")
	}
	return nil
}

func (p *ProductItem) validate() error {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.ImageURL = strings.TrimSpace(p.ImageURL)
	p.Status = strings.TrimSpace(p.Status)
	if p.Status == "" {
		p.Status = ProductStatusOnSale
	}
	if p.Status != ProductStatusOnSale && p.Status != ProductStatusOffSale {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "商品状态只能是 on_sale 或 off_sale")
	}
	if p.PriceCents < 0 || p.Stock < 0 {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "price_cents 和 stock 不能为负数")
	}
	return nil
}

func TrimMerchantPinnedProducts(tx *gorm.DB, merchantID uint) error {
	if merchantID == 0 {
		return nil
	}
	var products []Product
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND status = ? AND pinned_at IS NOT NULL", merchantID, ProductStatusOnSale).
		Order("pinned_at DESC, sort_order DESC, id DESC").
		Find(&products).Error; err != nil {
		return expect.Wrap(err, http.StatusInternalServerError, expect.CodeCommonInternalError, "查询置顶商品失败")
	}
	if len(products) <= maxMerchantPinnedProducts {
		return nil
	}
	dropIDs := make([]uint, 0, len(products)-maxMerchantPinnedProducts)
	for _, product := range products[maxMerchantPinnedProducts:] {
		dropIDs = append(dropIDs, product.Id)
	}
	db := tx.Model(&Product{}).Where("merchant_id = ? AND id IN ?", merchantID, dropIDs).UpdateColumn("pinned_at", nil)
	return expect.NDM(db, "清理置顶商品失败")
}

func RefreshAllProductSummaries(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&Merchant{}) || !db.Migrator().HasTable(&Product{}) || !db.Migrator().HasColumn(&Merchant{}, "products") {
		return nil
	}
	var merchants []Merchant
	if err := db.Select("id").Find(&merchants).Error; err != nil {
		return err
	}
	for _, merchant := range merchants {
		if err := RefreshProductSummaries(db, merchant.Id); err != nil {
			return err
		}
	}
	return nil
}

func RefreshProductSummaries(tx *gorm.DB, merchantID uint) error {
	return refreshProductSummaries(tx, merchantID, 0)
}

func refreshProductSummaries(tx *gorm.DB, merchantID uint, excludeProductID uint) error {
	if merchantID == 0 {
		return nil
	}
	products, err := merchantSummaryProducts(tx, merchantID, excludeProductID)
	if err != nil {
		return err
	}
	summaries := make([]ProductSummary, 0, len(products))
	for _, product := range products {
		summaries = append(summaries, ProductSummary{Name: product.Name, PriceCents: product.PriceCents})
	}
	raw, err := json.Marshal(summaries)
	if err != nil {
		return expect.Wrap(err, http.StatusInternalServerError, expect.CodeCommonInternalError, "序列化商户展示商品失败")
	}
	db := tx.Model(&Merchant{}).Where("id = ?", merchantID).UpdateColumn("products", string(raw))
	return expect.NDM(db, "刷新商户展示商品失败")
}

func merchantSummaryProducts(tx *gorm.DB, merchantID uint, excludeProductID uint) ([]Product, error) {
	var products []Product
	db := tx.
		Where("merchant_id = ? AND status = ? AND pinned_at IS NOT NULL", merchantID, ProductStatusOnSale).
		Order("pinned_at DESC, sort_order DESC, id DESC").
		Limit(maxMerchantPinnedProducts)
	if excludeProductID != 0 {
		db = db.Where("id <> ?", excludeProductID)
	}
	db = db.Find(&products)
	if db.Error != nil {
		return nil, db.Error
	}
	if len(products) >= maxMerchantPinnedProducts {
		return products, nil
	}
	excludeIDs := make([]uint, 0, len(products)+1)
	for _, product := range products {
		excludeIDs = append(excludeIDs, product.Id)
	}
	if excludeProductID != 0 {
		excludeIDs = append(excludeIDs, excludeProductID)
	}
	remain := maxMerchantPinnedProducts - len(products)
	var supplements []Product
	db = tx.
		Where("merchant_id = ? AND status = ?", merchantID, ProductStatusOnSale).
		Order("pinned_at IS NULL, pinned_at DESC, sort_order DESC, id DESC").
		Limit(remain)
	if len(excludeIDs) > 0 {
		db = db.Where("id NOT IN ?", excludeIDs)
	}
	db = db.Find(&supplements)
	if db.Error != nil {
		return nil, db.Error
	}
	return append(products, supplements...), nil
}

func refreshProductMerchantSummaries(tx *gorm.DB, productID uint, fallbackMerchantID uint) error {
	merchantID, err := productMerchantID(tx, productID, fallbackMerchantID)
	if err != nil || merchantID == 0 {
		return err
	}
	return RefreshProductSummaries(tx, merchantID)
}

func productMerchantID(tx *gorm.DB, productID uint, fallbackMerchantID uint) (uint, error) {
	if fallbackMerchantID != 0 {
		return fallbackMerchantID, nil
	}
	if productID == 0 {
		return 0, nil
	}
	var product Product
	if err := tx.Select("id", "merchant_id").First(&product, "id = ?", productID).Error; err != nil {
		return 0, err
	}
	return product.MerchantID, nil
}

func onSaleProductMerchantID(tx *gorm.DB, productID uint) (uint, error) {
	if productID == 0 {
		return 0, nil
	}
	var product Product
	if err := tx.Select("id", "merchant_id", "status").First(&product, "id = ?", productID).Error; err != nil {
		return 0, err
	}
	if product.Status != ProductStatusOnSale {
		db := tx.Model(&Product{}).Where("id = ?", productID).UpdateColumn("pinned_at", nil)
		if err := expect.NDM(db, "非上架商品取消置顶失败"); err != nil {
			return 0, err
		}
		return 0, expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "只有上架商品可以置顶")
	}
	return product.MerchantID, nil
}
