package service

import (
	"gkk/handler/jwts"

	bizmodel "github.com/gkk/stall-location/backend/internal/model"
	"github.com/gofiber/fiber/v3"
)

var Auth = jwts.Auth[bizmodel.User]
var User = jwts.User[bizmodel.User]

func UserScope(c fiber.Ctx) map[string]any {
	return map[string]any{"user_id": User(c).Id}
}

func MerchantAuth(c fiber.Ctx) error {
	if !User(c).IsMerchant() {
		return forbidden("当前登录身份不是商户")
	}
	return c.Next()
}
func MerchantScope(c fiber.Ctx) map[string]any {
	return map[string]any{"merchant_id": *User(c).MerchantID}
}
