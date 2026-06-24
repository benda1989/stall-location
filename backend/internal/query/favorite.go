package query

import (
	"strings"
	"time"

	gkkmodel "gkk/model"

	bizmodel "github.com/gkk/stall-location/backend/internal/model"
	"gorm.io/gorm"
)

type FavoriteQuery struct {
	gkkmodel.IdReq[uint]
	Name string `query:"name" find:"-"`
	gkkmodel.PageSize
}

func (q FavoriteQuery) DBS(db *gorm.DB) *gorm.DB {
	name := strings.TrimSpace(q.Name)
	if name != "" {
		like := "%" + name + "%"
		merchantIDs := db.Session(&gorm.Session{NewDB: true}).Model(&bizmodel.Merchant{}).Select("id").Where("display_name LIKE ?", like)
		db = db.Where("favorites.merchant_id IN (?)", merchantIDs)
	}
	return db.Preload("MerchantData").Preload("StallSessionData", "status = ? AND expected_end_at > ?", bizmodel.StatusActive, time.Now())
}

func (q FavoriteQuery) Order() string {
	return "favorites.id desc"
}
