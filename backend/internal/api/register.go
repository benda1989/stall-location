package api

import (
	"os"
	"path/filepath"

	"gkk/handler"
	uploadali "gkk/handler/upload/ali"
	"gkk/handler/urm"
	"gkk/handler/wx"

	"github.com/gkk/stall-location/backend/internal/conf"
	"github.com/gkk/stall-location/backend/internal/model"
	biz "github.com/gkk/stall-location/backend/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func Register(app *fiber.App, services *biz.Container) {
	wx.Register[model.User](false)

	api := app.Group("/api")
	api.Post("/upload", biz.Auth, uploadali.UploadFiber("user/nearby"))
	registerPublicRoutes(api.Group("/pub"), services)
	registerCustomerRoutes(api.Group("/customer", biz.Auth, handler.SetAuth(biz.UserScope)))
	registerMerchantRoutes(api.Group("/merchant", biz.Auth), services)
	registerFrontendRoutes(app)
	sys := urm.Register()
	registerSysRoutes(sys, services)
}

func registerFrontendRoutes(app *fiber.App) {

	indexPath := filepath.Join(conf.C.Frontend, "admin.html")
	if fileExists(indexPath) {
		app.Get("/daddy", func(c fiber.Ctx) error { return c.SendFile(indexPath) })
		app.Use("/assets", static.New(filepath.Join(conf.C.Frontend, "assets")))
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
