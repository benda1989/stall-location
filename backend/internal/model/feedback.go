package model

import (
	"net/http"
	"strings"
	"time"

	"gkk/expect"
	gkkmodel "gkk/model"

	"gorm.io/gorm"
)

type Feedback struct {
	FeedbackItem
	HandlerID   *uint      `json:"handler_id,omitempty" gorm:"index"`
	HandlerNote string     `json:"handler_note" gorm:"size:1024"`
	HandledAt   *time.Time `json:"handled_at,omitempty" gorm:"index"`
}

// FeedbackItem is the minimal create model for feedback. Feedback is the full
// read/handling model and embeds it for system processing fields.
type FeedbackItem struct {
	gkkmodel.IdReq[uint]
	gkkmodel.Timestamps
	Source       string `json:"source" gorm:"size:24;index;not null" validate:"required"`
	UserID       uint   `json:"-" gorm:"index;not null"`
	MerchantID   *uint  `json:"-" gorm:"index"`
	ContactName  string `json:"contact_name" gorm:"size:64"`
	ContactPhone string `json:"contact_phone" gorm:"size:64;index;not null" validate:"required"`
	Description  string `json:"description" gorm:"size:2048;not null" validate:"required"`
	ImageURL     string `json:"image_url" gorm:"type:text"`
	PageURL      string `json:"page_url" gorm:"size:512"`
	Status       string `json:"-" gorm:"size:24;index;default:pending"`
}

type FeedbackCreate = FeedbackItem

func (FeedbackItem) TableName() string { return "feedbacks" }

func (feedback *Feedback) BeforeSave(_ *gorm.DB) error {
	return normalizeFeedback(feedback)
}

func (feedback *FeedbackItem) BeforeSave(_ *gorm.DB) error {
	return normalizeFeedbackCreate(feedback)
}

func (feedback *FeedbackItem) BeforeCreate(_ *gorm.DB) error {
	switch feedback.Source {
	case FeedbackSourceCustomer:
		feedback.MerchantID = nil
	}
	return nil
}

func normalizeFeedback(feedback *Feedback) error {
	feedback.Source = strings.TrimSpace(feedback.Source)
	feedback.ContactName = strings.TrimSpace(feedback.ContactName)
	feedback.ContactPhone = strings.TrimSpace(feedback.ContactPhone)
	feedback.Description = strings.TrimSpace(feedback.Description)
	feedback.ImageURL = strings.TrimSpace(feedback.ImageURL)
	feedback.PageURL = strings.TrimSpace(feedback.PageURL)
	feedback.Status = strings.TrimSpace(feedback.Status)
	feedback.HandlerNote = strings.TrimSpace(feedback.HandlerNote)
	if feedback.Status == "" {
		feedback.Status = FeedbackStatusPending
	}
	if feedback.Source != "" && !isFeedbackSource(feedback.Source) {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "反馈来源不合法")
	}
	if !isFeedbackStatus(feedback.Status) {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "反馈状态不合法")
	}
	return nil
}

func normalizeFeedbackCreate(feedback *FeedbackItem) error {
	feedback.Source = strings.TrimSpace(feedback.Source)
	feedback.ContactName = strings.TrimSpace(feedback.ContactName)
	feedback.ContactPhone = strings.TrimSpace(feedback.ContactPhone)
	feedback.Description = strings.TrimSpace(feedback.Description)
	feedback.ImageURL = strings.TrimSpace(feedback.ImageURL)
	feedback.PageURL = strings.TrimSpace(feedback.PageURL)
	feedback.Status = FeedbackStatusPending
	if !isFeedbackSource(feedback.Source) {
		return expect.New(http.StatusUnprocessableEntity, expect.CodeCommonValidationError, "反馈来源不合法")
	}
	return nil
}

func isFeedbackSource(source string) bool {
	switch source {
	case FeedbackSourceCustomer, FeedbackSourceMerchant:
		return true
	default:
		return false
	}
}

func isFeedbackStatus(status string) bool {
	switch status {
	case FeedbackStatusPending, FeedbackStatusHandling, FeedbackStatusResolved, FeedbackStatusClosed:
		return true
	default:
		return false
	}
}
