package model

import (
	"strings"
	"time"

	gkkmodel "gkk/model"

	"github.com/gkk/stall-location/backend/internal/conf"
)

type PublicMerchant struct {
	Id           uint             `json:"id,omitempty"`
	DisplayName  string           `json:"display_name"`
	Category     string           `json:"category"`
	AvatarURL    string           `json:"avatar_url"`
	Announcement string           `json:"announcement"`
	Products     []ProductSummary `json:"products"`
}

type MerchantProfile struct {
	gkkmodel.Timestamps
	Id                 uint             `json:"id,omitempty"`
	DisplayName        string           `json:"display_name"`
	Phone              string           `json:"phone"`
	Category           string           `json:"category"`
	AvatarURL          string           `json:"avatar_url"`
	Announcement       string           `json:"announcement"`
	ContactPhone       string           `json:"contact_phone"`
	Products           []ProductSummary `json:"products"`
	ShareCode          string           `json:"share_code"`
	ShareURL           string           `json:"share_url"`
	SharePosterURL     string           `json:"share_poster_url"`
	ShareQRCodeURL     string           `json:"share_qrcode_url"`
	ShareQRCodeChannel string           `json:"share_qrcode_channel"`
	Status             string           `json:"status"`
	VerifyStatus       string           `json:"verify_status"`
	DisabledReason     string           `json:"disabled_reason,omitempty"`
}

type PublicStallSession struct {
	Status           string    `json:"status"`
	Lat              float64   `json:"lat"`
	Lng              float64   `json:"lng"`
	Address          string    `json:"address"`
	PhotoURL         string    `json:"photo_url"`
	LocationAccuracy int       `json:"location_accuracy"`
	StartedAt        time.Time `json:"started_at"`
	ExpectedEndAt    time.Time `json:"expected_end_at"`
}

type PublicProduct struct {
	Id          uint   `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents"`
	Stock       int    `json:"stock"`
	ImageURL    string `json:"image_url"`
}

func (merchant Merchant) Public() PublicMerchant {
	return PublicMerchant{
		Id:           merchant.Id,
		DisplayName:  merchant.DisplayName,
		Category:     merchant.Category,
		AvatarURL:    merchant.AvatarURL,
		Announcement: merchant.Announcement,
		Products:     merchant.Products,
	}
}

func (item MerchantItem) Json() any {
	return Merchant{MerchantItem: item}.Public()
}

func (merchant Merchant) Profile() MerchantProfile {
	posterURL := strings.TrimSpace(merchant.SharePosterURL)
	shareCode := strings.TrimSpace(merchant.ShareCode)
	return MerchantProfile{
		Id:                 merchant.Id,
		DisplayName:        merchant.DisplayName,
		Phone:              merchant.Phone,
		Category:           merchant.Category,
		AvatarURL:          merchant.AvatarURL,
		Announcement:       merchant.Announcement,
		ContactPhone:       merchant.ContactPhone,
		Products:           merchant.Products,
		ShareCode:          shareCode,
		ShareURL:           conf.ShareURL(shareCode),
		SharePosterURL:     posterURL,
		ShareQRCodeURL:     posterURL,
		ShareQRCodeChannel: merchant.ShareQRCodeChannel,
		Status:             merchant.Status,
		VerifyStatus:       merchant.VerifyStatus,
		DisabledReason:     merchant.DisabledReason,
		Timestamps:         merchant.Timestamps,
	}
}

func (merchant Merchant) Json() any {
	return merchant.Profile()
}

func (session StallSession) Public() PublicStallSession {
	return PublicStallSession{
		Status:           session.Status,
		Lat:              session.Lat,
		Lng:              session.Lng,
		Address:          session.Address,
		PhotoURL:         session.PhotoURL,
		LocationAccuracy: session.LocationAccuracy,
		StartedAt:        session.StartedAt,
		ExpectedEndAt:    session.ExpectedEndAt,
	}
}

func (session StallSession) Json() any {
	return session.Public()
}

func (product Product) Public() PublicProduct {
	return PublicProduct{
		Id:          product.Id,
		Name:        product.Name,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Stock:       product.Stock,
		ImageURL:    product.ImageURL,
	}
}

func PublicProducts(products []Product) []PublicProduct {
	items := make([]PublicProduct, 0, len(products))
	for _, product := range products {
		items = append(items, product.Public())
	}
	return items
}
