package model

import (
	"crypto/rand"
	"encoding/base32"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gkk/expect"
	gkkmodel "gkk/model"

	"gorm.io/gorm"
)

type ProductSummary struct {
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
}

// Merchant is the single business profile for a foreground merchant user.
type Merchant struct {
	MerchantItem
	gkkmodel.Timestamps
	ShareCode          string `json:"share_code" gorm:"size:32;uniqueIndex"`
	ShareURL           string `json:"share_url,omitempty" gorm:"-"`
	SharePosterURL     string `json:"share_poster_url,omitempty" gorm:"type:text"`
	ShareQRCodeURL     string `json:"share_qrcode_url,omitempty" gorm:"-"`
	ShareQRCodeChannel string `json:"share_qrcode_channel,omitempty" gorm:"column:share_qrcode_channel;size:32"`
	Status             string `json:"status" gorm:"size:24;index;default:active"`
	VerifyStatus       string `json:"verify_status" gorm:"size:24;index;default:unverified"`
	DisabledReason     string `json:"disabled_reason,omitempty" gorm:"size:512"`
}
type MerchantItem struct {
	gkkmodel.IdReq[uint]
	UserID       uint             `json:"-" gorm:"index;not null"`
	DisplayName  string           `json:"display_name" gorm:"size:120;index;not null"`
	Phone        string           `json:"phone" gorm:"size:32;index"`
	Category     string           `json:"category" gorm:"size:48;index"`
	AvatarURL    string           `json:"avatar_url" gorm:"size:512"`
	Announcement string           `json:"announcement" gorm:"size:512"`
	ContactPhone string           `json:"contact_phone" gorm:"size:32"`
	Products     []ProductSummary `json:"products,omitempty" gorm:"serializer:json"`
}

func (MerchantItem) TableName() string { return "merchants" }

func (item MerchantItem) Omit() string {
	return "id,user_id,share_code,share_poster_url,share_qrcode_channel,phone,products,status,verify_status,created_at"
}

type MerchantStatusUpdate struct {
	gkkmodel.IdReq[uint]
	Status         string `json:"status" gorm:"size:24;index;default:active" validate:"required"`
	DisabledReason string `json:"disabled_reason,omitempty" gorm:"size:512"`
}

func (MerchantStatusUpdate) TableName() string { return "merchants" }

func (status MerchantStatusUpdate) Omit() string { return "id" }

func (status *MerchantStatusUpdate) BeforeSave(_ *gorm.DB) error {
	status.Status = strings.TrimSpace(status.Status)
	status.DisabledReason = strings.TrimSpace(status.DisabledReason)
	if !isMerchantStatus(status.Status) {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "商户状态不合法")
	}
	if status.Status == StatusActive {
		status.DisabledReason = ""
	}
	return nil
}

func (merchant *Merchant) BeforeCreate(tx *gorm.DB) error {
	if err := merchant.BeforeSave(tx); err != nil {
		return err
	}
	if merchant.ShareCode == "" {
		merchant.ShareCode = nextMerchantShareCode()
	}
	return nil
}

func (merchant *Merchant) BeforeSave(_ *gorm.DB) error {
	merchant.ShareCode = strings.TrimSpace(merchant.ShareCode)
	merchant.SharePosterURL = strings.TrimSpace(merchant.SharePosterURL)
	merchant.ShareQRCodeURL = strings.TrimSpace(merchant.ShareQRCodeURL)
	merchant.ShareQRCodeChannel = strings.TrimSpace(merchant.ShareQRCodeChannel)
	merchant.DisplayName = strings.TrimSpace(merchant.DisplayName)
	merchant.Phone = strings.TrimSpace(merchant.Phone)
	merchant.Category = strings.TrimSpace(merchant.Category)
	merchant.AvatarURL = strings.TrimSpace(merchant.AvatarURL)
	merchant.Announcement = strings.TrimSpace(merchant.Announcement)
	merchant.ContactPhone = strings.TrimSpace(merchant.ContactPhone)
	merchant.Status = strings.TrimSpace(merchant.Status)
	merchant.VerifyStatus = strings.TrimSpace(merchant.VerifyStatus)
	merchant.DisabledReason = strings.TrimSpace(merchant.DisabledReason)
	if merchant.Status == "" {
		merchant.Status = StatusActive
	}
	if merchant.VerifyStatus == "" {
		merchant.VerifyStatus = VerifyUnverified
	}
	if !isMerchantStatus(merchant.Status) {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "商户状态不合法")
	}
	if !isVerifyStatus(merchant.VerifyStatus) {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "商户审核状态不合法")
	}
	return nil
}

func nextMerchantShareCode() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return strings.ToUpper("S" + strconv.FormatInt(time.Now().UnixNano(), 36))
	}
	code := strings.TrimRight(base32.StdEncoding.EncodeToString(buf), "=")
	if len(code) > 8 {
		return code[:8]
	}
	return code
}

func isMerchantStatus(status string) bool {
	switch status {
	case StatusActive, StatusDisabled:
		return true
	default:
		return false
	}
}

func isVerifyStatus(status string) bool {
	switch status {
	case VerifyUnverified, VerifyPending, VerifyVerified, VerifyRejected:
		return true
	default:
		return false
	}
}
