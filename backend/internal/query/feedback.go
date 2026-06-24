package query

import (
	gkkhandler "gkk/handler"
	gkkmodel "gkk/model"

	"gorm.io/gorm"
)

type FeedbackQuery struct {
	Source       string `query:"source" find:"="`
	UserID       uint   `query:"user_id" find:"="`
	MerchantID   uint   `query:"merchant_id" find:"="`
	ContactPhone string `query:"contact_phone" find:"="`
	Description  string `query:"description" find:"like"`
	Status       string `query:"status" find:"="`
	HandlerID    uint   `query:"handler_id" find:"="`
	gkkmodel.Period[uint]
}

func (q FeedbackQuery) DBS(db *gorm.DB) *gorm.DB {
	return gkkhandler.Encode2DB(q.Period.DBS(db), q)
}
