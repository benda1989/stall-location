package models

import (
	"time"

	"gorm.io/datatypes"
)

const (
	RoleCustomer = "customer"
	RoleMerchant = "merchant"
	RoleAdmin    = "admin"

	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"

	ShopStatusActive   = "active"
	ShopStatusDisabled = "disabled"

	VerifyUnverified = "unverified"
	VerifyPending    = "pending"
	VerifyVerified   = "verified"
	VerifyRejected   = "rejected"

	ProductStatusOnSale  = "on_sale"
	ProductStatusOffSale = "off_sale"

	StallStatusActive  = "active"
	StallStatusEnded   = "ended"
	StallStatusExpired = "expired"

	OrderPendingAccept = "pending_accept"
	OrderAccepted      = "accepted"
	OrderPreparing     = "preparing"
	OrderReady         = "ready"
	OrderCompleted     = "completed"
	OrderRejected      = "rejected"
	OrderCanceled      = "canceled"
	OrderExpired       = "expired"

	PaymentUnpaid    = "unpaid"
	PaymentPaying    = "paying"
	PaymentPaid      = "paid"
	PaymentRefunding = "refunding"
	PaymentRefunded  = "refunded"
	PaymentFailed    = "failed"

	ApplicationPending   = "pending"
	ApplicationReviewing = "reviewing"
	ApplicationNeedsInfo = "needs_info"
	ApplicationApproved  = "approved"
	ApplicationRejected  = "rejected"

	FeedbackSourceCustomer = "customer"
	FeedbackSourceMerchant = "merchant"
	FeedbackStatusPending  = "pending"
	FeedbackStatusHandling = "handling"
	FeedbackStatusResolved = "resolved"
	FeedbackStatusClosed   = "closed"
)

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Role      string    `json:"role" gorm:"size:24;index"`
	Phone     string    `json:"phone" gorm:"size:32;index"`
	OpenID    string    `json:"open_id,omitempty" gorm:"size:96;index"`
	Nickname  string    `json:"nickname" gorm:"size:96"`
	Status    string    `json:"status" gorm:"size:24;index;default:active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SystemRole struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	Code        string       `json:"code" gorm:"size:64;uniqueIndex"`
	Name        string       `json:"name" gorm:"size:96"`
	Description string       `json:"description" gorm:"size:255"`
	Status      string       `json:"status" gorm:"size:24;index;default:active"`
	SortOrder   int          `json:"sort_order"`
	Menus       []SystemMenu `json:"menus,omitempty" gorm:"many2many:system_role_menus;joinForeignKey:RoleID;joinReferences:MenuID"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type SystemMenu struct {
	ID         uint         `json:"id" gorm:"primaryKey"`
	ParentID   *uint        `json:"parent_id,omitempty" gorm:"index"`
	Code       string       `json:"code" gorm:"size:96;uniqueIndex"`
	Name       string       `json:"name" gorm:"size:96"`
	Path       string       `json:"path" gorm:"size:255"`
	Icon       string       `json:"icon" gorm:"size:64"`
	Type       string       `json:"type" gorm:"size:24;index"`
	Permission string       `json:"permission" gorm:"size:128;index"`
	Status     string       `json:"status" gorm:"size:24;index;default:active"`
	SortOrder  int          `json:"sort_order"`
	Children   []SystemMenu `json:"children,omitempty" gorm:"-"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

type SystemUserRole struct {
	UserID    uint      `json:"user_id" gorm:"primaryKey"`
	RoleID    uint      `json:"role_id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
}

type SystemRoleMenu struct {
	RoleID    uint      `json:"role_id" gorm:"primaryKey"`
	MenuID    uint      `json:"menu_id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
}

type Merchant struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"user_id" gorm:"index"`
	User        User      `json:"-"`
	DisplayName string    `json:"display_name" gorm:"size:96"`
	Phone       string    `json:"phone" gorm:"size:32;index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Shop struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	MerchantID     uint      `json:"merchant_id" gorm:"index"`
	Merchant       Merchant  `json:"-"`
	ShopCode       string    `json:"shop_code" gorm:"size:32;uniqueIndex"`
	Name           string    `json:"name" gorm:"size:120"`
	Category       string    `json:"category" gorm:"size:48"`
	AvatarURL      string    `json:"avatar_url" gorm:"size:512"`
	Announcement   string    `json:"announcement" gorm:"size:512"`
	ContactPhone   string    `json:"contact_phone" gorm:"size:32"`
	Status         string    `json:"status" gorm:"size:24;index"`
	VerifiedStatus string    `json:"verified_status" gorm:"size:24;index"`
	DisabledReason string    `json:"disabled_reason,omitempty" gorm:"size:512"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Product struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ShopID      uint      `json:"shop_id" gorm:"index"`
	Shop        Shop      `json:"-"`
	Name        string    `json:"name" gorm:"size:120"`
	Description string    `json:"description" gorm:"size:512"`
	PriceCents  int64     `json:"price_cents"`
	Stock       int       `json:"stock"`
	ImageURL    string    `json:"image_url" gorm:"size:512"`
	Status      string    `json:"status" gorm:"size:24;index"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StallSession struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	ShopID           uint       `json:"shop_id" gorm:"index"`
	Shop             Shop       `json:"-"`
	Status           string     `json:"status" gorm:"size:24;index"`
	Lat              float64    `json:"lat"`
	Lng              float64    `json:"lng"`
	Address          string     `json:"address" gorm:"size:255"`
	LocationAccuracy int        `json:"location_accuracy"`
	StartedAt        time.Time  `json:"started_at"`
	ExpectedEndAt    time.Time  `json:"expected_end_at"`
	EndedAt          *time.Time `json:"ended_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type Order struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	OrderNo          string         `json:"order_no" gorm:"size:32;uniqueIndex"`
	ShopID           uint           `json:"shop_id" gorm:"index"`
	Shop             Shop           `json:"shop"`
	StallSessionID   uint           `json:"stall_session_id" gorm:"index"`
	StallSession     StallSession   `json:"-"`
	CustomerID       *uint          `json:"customer_id" gorm:"index"`
	CustomerName     string         `json:"customer_name" gorm:"size:64"`
	CustomerPhone    string         `json:"customer_phone" gorm:"size:32"`
	PickupCode       string         `json:"pickup_code" gorm:"size:12"`
	PickupTime       *time.Time     `json:"pickup_time"`
	Status           string         `json:"status" gorm:"size:32;index"`
	PaymentStatus    string         `json:"payment_status" gorm:"size:32;index"`
	TotalAmountCents int64          `json:"total_amount_cents"`
	Remark           string         `json:"remark" gorm:"size:512"`
	LocationSnapshot datatypes.JSON `json:"location_snapshot"`
	Items            []OrderItem    `json:"items"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type OrderItem struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	OrderID        uint      `json:"order_id" gorm:"index"`
	ProductID      uint      `json:"product_id" gorm:"index"`
	ProductName    string    `json:"product_name" gorm:"size:120"`
	UnitPriceCents int64     `json:"unit_price_cents"`
	Quantity       int       `json:"quantity"`
	SubtotalCents  int64     `json:"subtotal_cents"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ShopQRCode struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ShopID    uint      `json:"shop_id" gorm:"index"`
	Shop      Shop      `json:"-"`
	CodeType  string    `json:"code_type" gorm:"size:24"`
	URL       string    `json:"url" gorm:"size:512"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MerchantApplication struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	ApplicationNo string     `json:"application_no" gorm:"size:32;uniqueIndex"`
	ShopName      string     `json:"shop_name" gorm:"size:120;index"`
	ContactName   string     `json:"contact_name" gorm:"size:64"`
	ContactPhone  string     `json:"contact_phone" gorm:"size:32;index"`
	Category      string     `json:"category" gorm:"size:48;index"`
	PhotoURL      string     `json:"photo_url" gorm:"size:2048"`
	UsualArea     string     `json:"usual_area" gorm:"size:255"`
	Remark        string     `json:"remark" gorm:"size:512"`
	Status        string     `json:"status" gorm:"size:24;index"`
	ReviewReason  string     `json:"review_reason,omitempty" gorm:"size:1024"`
	ReviewerID    *uint      `json:"reviewer_id,omitempty" gorm:"index"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	MerchantID    *uint      `json:"merchant_id,omitempty" gorm:"index"`
	ShopID        *uint      `json:"shop_id,omitempty" gorm:"index"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type SMSCode struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	Phone     string     `json:"phone" gorm:"size:32;index"`
	Scene     string     `json:"scene" gorm:"size:24;index"`
	Code      string     `json:"-" gorm:"size:12"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"index"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Feedback struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	Source       string     `json:"source" gorm:"size:24;index"`
	UserID       *uint      `json:"user_id,omitempty" gorm:"index"`
	User         *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	ShopID       *uint      `json:"shop_id,omitempty" gorm:"index"`
	Shop         *Shop      `json:"shop,omitempty" gorm:"foreignKey:ShopID"`
	ContactName  string     `json:"contact_name" gorm:"size:64"`
	ContactPhone string     `json:"contact_phone" gorm:"size:64;index"`
	Description  string     `json:"description" gorm:"size:2048"`
	ImageURL     string     `json:"image_url" gorm:"type:text"`
	PageURL      string     `json:"page_url" gorm:"size:512"`
	Status       string     `json:"status" gorm:"size:24;index;default:pending"`
	HandlerID    *uint      `json:"handler_id,omitempty" gorm:"index"`
	HandlerNote  string     `json:"handler_note" gorm:"size:1024"`
	HandledAt    *time.Time `json:"handled_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
