package model

import (
	"net/http"

	"gkk/expect"
	gkkmodel "gkk/model"

	"github.com/gkk/stall-location/backend/internal/conf"
	"gorm.io/gorm"
)

type Favorite struct {
	gkkmodel.IdReq[uint]
	gkkmodel.Timestamps
	UserID           uint                `json:"-" gorm:"uniqueIndex:idx_customer_favorites_customer_merchant;not null"`
	MerchantID       uint                `json:"merchant_id" gorm:"uniqueIndex:idx_customer_favorites_customer_merchant;not null"`
	MerchantData     *Merchant           `json:"-" gorm:"foreignKey:MerchantID"`
	Merchant         *PublicMerchant     `json:"merchant,omitempty" gorm:"-"`
	StallSessionData *StallSession       `json:"-" gorm:"foreignKey:MerchantID;references:MerchantID"`
	StallSession     *PublicStallSession `json:"stall_session,omitempty" gorm:"-"`
}

func (Favorite) TableName() string { return "favorites" }

func (favorite *Favorite) AfterFind(_ *gorm.DB) error {
	if favorite == nil {
		return nil
	}
	if favorite.MerchantData != nil {
		public := favorite.MerchantData.Public()
		favorite.Merchant = &public
	}
	if favorite.StallSessionData != nil {
		public := favorite.StallSessionData.Public()
		favorite.StallSession = &public
	}
	return nil
}

func (favorite *Favorite) BeforeCreate(tx *gorm.DB) error {
	limit := conf.MaxCustomerFavorites()
	if favorite.UserID == 0 || limit <= 0 {
		return nil
	}
	var total int64
	if err := tx.Model(&Favorite{}).Where("user_id = ?", favorite.UserID).Count(&total).Error; err != nil {
		return expect.Wrap(err, http.StatusInternalServerError, expect.CodeCommonInternalError, "查询顾客收藏数量失败")
	}
	if total >= int64(limit) {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "顾客收藏数量已达上限")
	}
	return nil
}
