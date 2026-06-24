package api

import (
	"gkk/handler"
	"gkk/model"

	bizmodel "github.com/gkk/stall-location/backend/internal/model"
	bizquery "github.com/gkk/stall-location/backend/internal/query"
	biz "github.com/gkk/stall-location/backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

func registerMerchantRoutes(merchant fiber.Router, services *biz.Container) {
	merchant.Use(biz.MerchantAuth)
	profileScope := handler.SetAuth(func(c fiber.Ctx) map[string]any {
		return map[string]any{"id": *biz.User(c).MerchantID}
	})
	merchant.Get("/me", profileScope, handler.First[model.IdReq[uint], bizmodel.Merchant])
	merchant.Put("/me", profileScope, handler.UpdateOnly[bizmodel.MerchantItem])

	merchant.Use(handler.SetAuth(biz.MerchantScope))
	{
		merchant.Get("/stalls", handler.Get[bizquery.StallSessionQuery, bizmodel.StallSession])
		handler.PostJSON(merchant, "/stalls/start", services.Stall.Start)
		handler.Post(merchant, "/stalls/end", services.Stall.End)
		handler.GetList[bizquery.ProductQuery, bizmodel.Product](merchant, "/products")
		merchant.Put("/products", handler.Update[bizmodel.ProductItem])
		merchant.Delete("/products", handler.Delete[bizmodel.Product])
		merchant.Put("/products/pin", handler.UpdateOnly[bizmodel.ProductPinUpdate])
		merchant.Put("/products/unpin", handler.UpdateOnly[bizmodel.ProductUnpinUpdate])
	}
}
