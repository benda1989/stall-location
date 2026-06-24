package model

import (
	"time"

	"gkk/expect"
	gkkmodel "gkk/model"

	"gorm.io/gorm"
)

type Application struct {
	ApplicationItem
	ReviewReason string     `json:"review_reason,omitempty" gorm:"size:1024"`
	ReviewerID   *uint      `json:"reviewer_id,omitempty" gorm:"index"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty" gorm:"index"`
	MerchantID   *uint      `json:"merchant_id,omitempty" gorm:"index"`
	Merchant     *Merchant  `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
}

type ApplicationItem struct {
	gkkmodel.IdReq[uint]
	gkkmodel.Timestamps
	UserID       uint   `json:"-" gorm:"index;not null"`
	MerchantName string `json:"merchant_name" gorm:"size:120;index;not null" validate:"required"`
	ContactName  string `json:"contact_name" gorm:"size:64;not null" validate:"required"`
	ContactPhone string `json:"contact_phone" gorm:"size:32;index;not null" validate:"required"`
	Category     string `json:"category" gorm:"size:48;index" validate:"required"`
	PhotoURL     string `json:"photo_url" gorm:"size:2048" validate:"required"`
	UsualArea    string `json:"usual_area" gorm:"size:255" validate:"required"`
	Remark       string `json:"remark" gorm:"size:512" validate:"required"`
	Status       string `json:"status" gorm:"size:24;index;default:pending"`
}

func (ApplicationItem) TableName() string { return "applications" }

func (app ApplicationItem) Omit() string {
	return "id,user_id,status,created_at"
}

func (app *ApplicationItem) BeforeCreate(tx *gorm.DB) error {
	if app.Status == "" {
		app.Status = ApplicationPending
	}
	return expect.NDM(tx.Model(&User{}).Where("id = ?", app.UserID).UpdateColumn("page_mode", IdentityApplication), "更新用户申请状态失败")
}
