package model

import (
	"net/http"
	"strings"
	"time"

	"gkk/expect"
	gkkmodel "gkk/model"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Preorder struct {
	gkkmodel.IdReq[uint]
	gkkmodel.Timestamps
	OrderNo          string         `json:"order_no" gorm:"size:32;uniqueIndex;not null"`
	MerchantID       uint           `json:"merchant_id" gorm:"index;not null"`
	Merchant         *Merchant      `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
	StallSessionID   uint           `json:"stall_session_id" gorm:"index;not null"`
	StallSession     *StallSession  `json:"stall_session,omitempty" gorm:"foreignKey:StallSessionID"`
	UserID           uint           `json:"-" gorm:"index;not null"`
	CustomerName     string         `json:"customer_name" gorm:"size:64;not null"`
	CustomerPhone    string         `json:"customer_phone" gorm:"size:32;index;not null"`
	PickupCode       string         `json:"pickup_code" gorm:"size:12;index;not null"`
	PickupTime       *time.Time     `json:"pickup_time,omitempty" gorm:"index"`
	Status           string         `json:"status" gorm:"size:32;index;default:pending_accept"`
	PaymentStatus    string         `json:"payment_status" gorm:"size:32;index;default:unpaid"`
	TotalAmountCents int64          `json:"total_amount_cents" gorm:"not null;default:0"`
	Remark           string         `json:"remark" gorm:"size:512"`
	LocationSnapshot datatypes.JSON `json:"location_snapshot" gorm:"type:json"`
	Items            []PreorderItem `json:"items,omitempty" gorm:"foreignKey:OrderID"`
}

type PreorderItem struct {
	gkkmodel.IdReq[uint]
	gkkmodel.Timestamps
	OrderID        uint   `json:"order_id" gorm:"index;not null"`
	ProductID      uint   `json:"product_id" gorm:"index;not null"`
	ProductName    string `json:"product_name" gorm:"size:120;not null"`
	UnitPriceCents int64  `json:"unit_price_cents" gorm:"not null;default:0"`
	Quantity       int    `json:"quantity" gorm:"not null;default:1"`
	SubtotalCents  int64  `json:"subtotal_cents" gorm:"not null;default:0"`
}

func (order *Preorder) BeforeSave(_ *gorm.DB) error {
	order.OrderNo = strings.TrimSpace(order.OrderNo)
	order.CustomerName = strings.TrimSpace(order.CustomerName)
	order.CustomerPhone = strings.TrimSpace(order.CustomerPhone)
	order.PickupCode = strings.TrimSpace(order.PickupCode)
	order.Status = strings.TrimSpace(order.Status)
	order.PaymentStatus = strings.TrimSpace(order.PaymentStatus)
	order.Remark = strings.TrimSpace(order.Remark)
	if order.Status == "" {
		order.Status = OrderPendingAccept
	}
	if order.PaymentStatus == "" {
		order.PaymentStatus = PaymentUnpaid
	}
	if !isOrderStatus(order.Status) {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "订单状态不合法")
	}
	if !isPaymentStatus(order.PaymentStatus) {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "支付状态不合法")
	}
	if order.TotalAmountCents < 0 {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "total_amount_cents 不能为负数")
	}
	return nil
}

func (item *PreorderItem) BeforeSave(_ *gorm.DB) error {
	item.ProductName = strings.TrimSpace(item.ProductName)
	if item.Quantity <= 0 {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "quantity 必须大于 0")
	}
	if item.UnitPriceCents < 0 || item.SubtotalCents < 0 {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "金额不能为负数")
	}
	return nil
}

func (item *PreorderItem) BeforeCreate(_ *gorm.DB) error {
	if item.OrderID == 0 || item.ProductID == 0 || item.ProductName == "" {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "order_id、product_id、product_name 必填")
	}
	return nil
}

func isOrderStatus(status string) bool {
	switch status {
	case OrderPendingAccept, OrderAccepted, OrderPreparing, OrderReady, OrderCompleted, OrderRejected, OrderCanceled, OrderExpired:
		return true
	default:
		return false
	}
}

func isPaymentStatus(status string) bool {
	switch status {
	case PaymentUnpaid, PaymentPaying, PaymentPaid, PaymentRefunding, PaymentRefunded, PaymentFailed:
		return true
	default:
		return false
	}
}
