package model

import (
	"net/http"
	"time"

	"gkk/expect"
	gkkmodel "gkk/model"

	"gorm.io/gorm"
)

type StallSession struct {
	gkkmodel.IdReq[uint]
	gkkmodel.Timestamps
	MerchantID       uint       `json:"merchant_id" gorm:"index;not null"`
	Merchant         *Merchant  `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
	Status           string     `json:"status" gorm:"size:24;index;default:active"`
	Lat              float64    `json:"lat" gorm:"index"  validate:"required"`
	Lng              float64    `json:"lng" gorm:"index"  validate:"required"`
	Address          string     `json:"address" gorm:"size:255"  validate:"required"`
	PhotoURL         string     `json:"photo_url" gorm:"size:2048"  validate:"required"`
	LocationAccuracy int        `json:"location_accuracy" gorm:"default:0"  validate:"required"`
	StartedAt        time.Time  `json:"started_at" gorm:"index;not null"  validate:"required"`
	ExpectedEndAt    time.Time  `json:"expected_end_at" gorm:"index;not null" validate:"required"`
	EndedAt          *time.Time `json:"ended_at,omitempty" gorm:"index" `
}

func (session *StallSession) BeforeSave(_ *gorm.DB) error {
	if session.Status == StatusActive {
		session.Status = StatusEnded
	}
	if session.LocationAccuracy < 0 {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "location_accuracy 不能为负数")
	}
	return nil
}

func (session *StallSession) BeforeCreate(_ *gorm.DB) error {
	session.Status = StatusActive
	if !session.ExpectedEndAt.After(session.StartedAt) {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "expected_end_at 必须晚于 started_at")
	}
	return nil
}
