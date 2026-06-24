package bootstrap

import (
	"gkk/orm"

	"github.com/gkk/stall-location/backend/internal/api"
	biz "github.com/gkk/stall-location/backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

// Register wires concrete business services before handing route registration to api.
func Register(app *fiber.App) {
	api.Register(app, biz.NewContainer(orm.DB))
}
