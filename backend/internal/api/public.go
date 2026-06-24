package api

import (
	"gkk/handler"

	bizmodel "github.com/gkk/stall-location/backend/internal/model"
	bizquery "github.com/gkk/stall-location/backend/internal/query"
	biz "github.com/gkk/stall-location/backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

func registerPublicRoutes(api fiber.Router, services *biz.Container) {
	handler.GetQuery(api, "/stalls/nearby", services.Stall.NearbyAction)
	api.Get("/merchants/detail", handler.First[bizquery.PublicMerchantQuery, bizmodel.MerchantItem])
	api.Get("/merchants/stall", handler.OptionalFirst[bizquery.PublicActiveStallSessionQuery, bizmodel.StallSession])
	handler.GetQuery(api, "/merchants/products", services.Stall.PublicProductsAction)
}
