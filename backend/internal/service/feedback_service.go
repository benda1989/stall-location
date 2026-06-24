package service

import (
	"strings"

	"gkk/expect"
	"gkk/handler/jwts"
	"gkk/orm"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FeedbackService struct {
	DB  *gorm.DB
	Now clock
}

type UpdateFeedbackRequest struct {
	Status      string `json:"status" validate:"required"`
	HandlerNote string `json:"handler_note"`
}

func (s *FeedbackService) UpdateSys(c fiber.Ctx, id uint, req UpdateFeedbackRequest) (any, error) {
	operatorID := jwts.UserInfo(c).Id
	if !validFeedbackStatus(req.Status) {
		return nil, validation("status 必须是 pending/handling/resolved/closed")
	}
	return nil, withTx(s.DB, func(tx *gorm.DB) error {
		feedback := orm.First[Feedback](tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id))
		if feedback.Id == 0 {
			return notFound("反馈不存在")
		}
		patch := Feedback{
			FeedbackItem: FeedbackItem{Status: req.Status},
			HandlerID:    ptrUint(operatorID),
			HandlerNote:  strings.TrimSpace(req.HandlerNote),
		}
		if req.Status == FeedbackStatusResolved || req.Status == FeedbackStatusClosed {
			patch.HandledAt = ptrTime(serviceNow(s.Now))
		} else {
			patch.HandledAt = nil
		}
		fields := []string{"status", "handler_id", "handler_note", "handled_at"}
		return expect.NDM(tx.Model(&Feedback{}).Where("id = ?", feedback.Id).Select(fields).Updates(patch), "更新反馈失败")
	})
}

func validFeedbackStatus(status string) bool {
	switch status {
	case FeedbackStatusPending, FeedbackStatusHandling, FeedbackStatusResolved, FeedbackStatusClosed:
		return true
	default:
		return false
	}
}
