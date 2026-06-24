package model

import (
	gkkuser "gkk/model/user"
	"gkk/orm"
)

type User struct {
	gkkuser.Info
	MerchantID *uint  `json:"merchant_id,omitempty"`
	PageMode   string `json:"page_mode" gorm:"size:24;index;default:customer"`
}

func (u User) Disable() { u.Info.Disable(&u) }
func (u User) Create() any {
	u.PageMode = IdentityCustomer
	u.IsValid = true
	orm.DB.Create(&u)
	return u
}

func (u User) IsMerchant() bool {
	return u.MerchantID != nil
}
