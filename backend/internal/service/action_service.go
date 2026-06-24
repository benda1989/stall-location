package service

import (
	"gkk/expect"
	"gkk/handler/jwts"

	"github.com/gofiber/fiber/v3"
)

func (s *PreorderService) MerchantOrderAction(action string) func(fiber.Ctx, uint) (any, error) {
	return func(c fiber.Ctx, id uint) (any, error) {
		merchantID := *User(c).MerchantID
		switch action {
		case "accept":
			return nil, s.merchantTransition(merchantID, id, OrderAccepted, false)
		case "reject":
			return nil, s.merchantTransition(merchantID, id, OrderRejected, true)
		case "prepare":
			return nil, s.merchantTransition(merchantID, id, OrderPreparing, false)
		case "ready":
			return nil, s.merchantTransition(merchantID, id, OrderReady, false)
		case "complete":
			return nil, s.merchantTransition(merchantID, id, OrderCompleted, false)
		default:
			return nil, badRequest(expect.CodeCommonBadRequest, "未知订单动作")
		}
	}
}

func (s *ApplicationService) ReviewApplicationAction(action string) func(fiber.Ctx, uint, ReviewApplicationRequest) (any, error) {
	return func(c fiber.Ctx, id uint, req ReviewApplicationRequest) (any, error) {
		switch action {
		case "approve":
			return nil, s.Approve(jwts.UserInfo(c), id, req.ReviewReason)
		case "reject":
			return nil, s.Reject(jwts.UserInfo(c), id, req.ReviewReason)
		default:
			return nil, badRequest(expect.CodeCommonBadRequest, "未知审核动作")
		}
	}
}
