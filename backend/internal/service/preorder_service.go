package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gkk/expect"
	"gkk/orm"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PreorderService struct {
	DB  *gorm.DB
	Now clock
}

type CreatePreorderRequest struct {
	CustomerName  string                  `json:"customer_name" validate:"required"`
	CustomerPhone string                  `json:"customer_phone" validate:"required"`
	PickupTime    *time.Time              `json:"pickup_time"`
	Remark        string                  `json:"remark"`
	Items         []CreatePreorderItemReq `json:"items" validate:"required,min=1,dive"`
}

type CreatePreorderItemReq struct {
	ProductID uint `json:"product_id" validate:"required"`
	Quantity  int  `json:"quantity" validate:"required,gt=0"`
}

type locationSnapshot struct {
	SessionID  uint      `json:"session_id"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Address    string    `json:"address"`
	PhotoURL   string    `json:"photo_url"`
	CapturedAt time.Time `json:"captured_at"`
}

func (s *PreorderService) Create(c fiber.Ctx, req CreatePreorderRequest) (any, error) {
	userID := User(c).Id
	customerName := strings.TrimSpace(req.CustomerName)
	customerPhone := strings.TrimSpace(req.CustomerPhone)
	return nil, withTx(s.DB, func(tx *gorm.DB) error {
		firstItem := req.Items[0]
		firstProduct := orm.First[Product](tx.Where("id = ? AND status = ?", firstItem.ProductID, ProductStatusOnSale))
		if firstProduct.Id == 0 {
			return notFound("商品不存在或已下架")
		}
		merchant := orm.First[Merchant](tx.Where("id = ? AND status = ? AND verify_status = ?", firstProduct.MerchantID, StatusActive, VerifyVerified))
		if merchant.Id == 0 {
			return notFound("商户不存在或已停用")
		}
		session := orm.First[StallSession](tx.Where("merchant_id = ? AND status = ? AND expected_end_at > ?", merchant.Id, StatusActive, serviceNow(s.Now)).Order("started_at DESC"))
		if session.Id == 0 {
			return notFound("商户当前未出摊")
		}
		snapshot, err := json.Marshal(locationSnapshot{SessionID: session.Id, Lat: session.Lat, Lng: session.Lng, Address: session.Address, PhotoURL: session.PhotoURL, CapturedAt: serviceNow(s.Now)})
		if err != nil {
			return expect.Wrap(err, 500, expect.CodeCommonInternalError, "生成位置快照失败")
		}
		order := Preorder{OrderNo: s.nextOrderNo(), MerchantID: merchant.Id, StallSessionID: session.Id, UserID: userID, CustomerName: customerName, CustomerPhone: customerPhone, PickupCode: randomCode("", 4), PickupTime: req.PickupTime, Status: OrderPendingAccept, PaymentStatus: PaymentUnpaid, Remark: strings.TrimSpace(req.Remark), LocationSnapshot: snapshot}
		if db := tx.Create(&order); db.Error != nil {
			return expect.NDM(db, "创建订单失败")
		}
		var total int64
		for _, itemReq := range req.Items {
			product := orm.First[Product](tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ? AND status = ?", itemReq.ProductID, merchant.Id, ProductStatusOnSale))
			if product.Id == 0 {
				return notFound("商品不存在或已下架")
			}
			if product.Stock < itemReq.Quantity {
				return conflict(expect.CodeOrderStatusInvalid, fmt.Sprintf("%s 库存不足", product.Name))
			}
			db := tx.Model(&Product{}).Where("id = ? AND merchant_id = ? AND stock >= ?", product.Id, merchant.Id, itemReq.Quantity).UpdateColumn("stock", gorm.Expr("stock - ?", itemReq.Quantity))
			if db.Error != nil {
				return expect.NDM(db, "扣减库存失败")
			}
			if db.RowsAffected == 0 {
				return conflict(expect.CodeOrderStatusInvalid, fmt.Sprintf("%s 库存不足", product.Name))
			}
			subtotal := product.PriceCents * int64(itemReq.Quantity)
			orderItem := PreorderItem{OrderID: order.Id, ProductID: product.Id, ProductName: product.Name, UnitPriceCents: product.PriceCents, Quantity: itemReq.Quantity, SubtotalCents: subtotal}
			if db := tx.Create(&orderItem); db.Error != nil {
				return expect.NDM(db, "创建订单明细失败")
			}
			total += subtotal
		}
		if db := tx.Model(&Preorder{}).Where("id = ? AND user_id = ? AND merchant_id = ?", order.Id, userID, merchant.Id).Update("total_amount_cents", total); db.Error != nil {
			return expect.NDM(db, "更新订单金额失败")
		}
		return nil
	})
}

func (s *PreorderService) CustomerCancel(c fiber.Ctx, orderNo string) (any, error) {
	userID := User(c).Id
	return nil, withTx(s.DB, func(tx *gorm.DB) error {
		order := orm.First[Preorder](tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_no = ? AND user_id = ?", strings.TrimSpace(orderNo), userID))
		if order.Id == 0 {
			return notFound("订单不存在")
		}
		return s.transitionPreorder(tx, &order, OrderCanceled, true, func(db *gorm.DB) *gorm.DB {
			return db.Where("user_id = ?", userID)
		})
	})
}

func (s *PreorderService) SysCancel(c fiber.Ctx, id uint) (any, error) {
	return nil, withTx(s.DB, func(tx *gorm.DB) error {
		order := orm.First[Preorder](tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id))
		if order.Id == 0 {
			return notFound("订单不存在")
		}
		if err := s.transitionPreorder(tx, &order, OrderCanceled, true, nil); err != nil {
			return err
		}
		return nil
	})
}

func (s *PreorderService) SysRefund(c fiber.Ctx, id uint) (any, error) {
	return nil, withTx(s.DB, func(tx *gorm.DB) error {
		order := orm.First[Preorder](tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id))
		if order.Id == 0 {
			return notFound("订单不存在")
		}
		if order.PaymentStatus == PaymentRefunded {
			return nil
		}
		if order.PaymentStatus != PaymentPaid && order.PaymentStatus != PaymentRefunding {
			return conflict(expect.CodeOrderStatusInvalid, "当前支付状态不能标记退款")
		}
		return expect.NDM(tx.Model(&Preorder{}).Where("id = ?", order.Id).Update("payment_status", PaymentRefunded), "退款标记失败")
	})
}

func (s *PreorderService) merchantTransition(merchantID uint, id uint, to string, restock bool) error {
	return withTx(s.DB, func(tx *gorm.DB) error {
		order := orm.First[Preorder](tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", id, merchantID))
		if order.Id == 0 {
			return notFound("订单不存在")
		}
		return s.transitionPreorder(tx, &order, to, restock, func(db *gorm.DB) *gorm.DB {
			return db.Where("merchant_id = ?", merchantID)
		})
	})
}

func (s *PreorderService) transitionPreorder(tx *gorm.DB, order *Preorder, to string, restock bool, scope func(*gorm.DB) *gorm.DB) error {
	if !allowedOrderTransition(order.Status, to) {
		return conflict(expect.CodeOrderStatusInvalid, "订单状态不允许该操作")
	}
	if restock {
		if err := s.restockOrder(tx, order); err != nil {
			return err
		}
	}
	db := tx.Model(&Preorder{}).Where("id = ?", order.Id)
	if scope != nil {
		db = scope(db)
	}
	return expect.NDM(db.Update("status", to), "订单状态更新失败")
}

func allowedOrderTransition(from, to string) bool {
	switch to {
	case OrderAccepted:
		return from == OrderPendingAccept
	case OrderPreparing:
		return from == OrderAccepted
	case OrderReady:
		return from == OrderAccepted || from == OrderPreparing
	case OrderCompleted:
		return from == OrderReady
	case OrderRejected:
		return from == OrderPendingAccept
	case OrderCanceled:
		return from == OrderPendingAccept || from == OrderAccepted || from == OrderPreparing
	default:
		return false
	}
}

func (s *PreorderService) restockOrder(tx *gorm.DB, order *Preorder) error {
	items := orm.Find[PreorderItem](tx.Where("order_id = ?", order.Id))
	for _, item := range items {
		if db := tx.Model(&Product{}).Where("id = ? AND merchant_id = ?", item.ProductID, order.MerchantID).UpdateColumn("stock", gorm.Expr("stock + ?", item.Quantity)); db.Error != nil {
			return expect.NDM(db, "回补库存失败")
		}
	}
	return nil
}

func (s *PreorderService) nextOrderNo() string {
	return fmt.Sprintf("PO%s%s", serviceNow(s.Now).Format("20060102150405"), randomCode("", 6))
}
