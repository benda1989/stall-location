package api

import (
	"gkk/handler"
	"gkk/model"

	bizmodel "github.com/gkk/stall-location/backend/internal/model"
	bizquery "github.com/gkk/stall-location/backend/internal/query"
	biz "github.com/gkk/stall-location/backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

func registerCustomerRoutes(api fiber.Router) {
	api.Get("/me", func(c fiber.Ctx) error { return c.JSON(biz.User(c)) })
	api.Get("/applications", handler.First[model.IdReq[uint], bizmodel.Application])
	api.Put("/applications", handler.Update[bizmodel.ApplicationItem])
	api.Post("/feedback", handler.Create[bizmodel.FeedbackItem])
	handler.GetList[bizquery.FavoriteQuery, bizmodel.Favorite](api, "/favorites")
	handler.PostJSON(api, "/favorites", handler.FirstOrCreate[bizmodel.Favorite]("添加收藏失败"))
	api.Delete("/favorites", handler.Delete[bizmodel.Favorite, uint])
}
