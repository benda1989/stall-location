package api

import (
	"path/filepath"
	"strings"

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
	root := strings.TrimSpace(conf.C.Frontend)
	if root == "" {
		return
	}
	indexPath := filepath.Join(root, "index.html")
	app.Use("/", static.New(root, static.Config{
		Next: func(c fiber.Ctx) bool {
			return skipFrontendRoute(c)
		},
		NotFoundHandler: func(c fiber.Ctx) error {
			if !frontendFallbackRequest(c) {
				return c.Next()
			}
			return c.SendFile(indexPath)
		},
	}))
}

func skipFrontendRoute(c fiber.Ctx) bool {
	method := c.Method()
	if method != fiber.MethodGet && method != fiber.MethodHead {
		return true
	}
	path := c.Path()
	return path == "/healthz" ||
		strings.HasPrefix(path, "/api")
}

func frontendFallbackRequest(c fiber.Ctx) bool {
	ext := strings.ToLower(filepath.Ext(c.Path()))
	if ext != "" && ext != ".html" {
		return false
	}
	accept := c.Get("Accept")
	return accept == "" || strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*")
}
