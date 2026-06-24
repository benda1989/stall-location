package api

import (
	"gkk/handler"
	"gkk/handler/jwts"

	bizmodel "github.com/gkk/stall-location/backend/internal/model"
	bizquery "github.com/gkk/stall-location/backend/internal/query"
	biz "github.com/gkk/stall-location/backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

func registerSysRoutes(sys fiber.Router, services *biz.Container) {
	business := sys.Group("", jwts.Role).Name("业务后台")

	handler.GetList[bizquery.ApplicationQuery, bizmodel.Application](business, "/applications").Name(".入驻申请.查看列表")
	business.Get("/applications/detail", handler.First[bizquery.ApplicationDetailQuery, bizmodel.Application]).Name(".入驻申请.查看详情")
	applicationActionNames := map[string]string{"approve": "审核通过", "reject": "审核拒绝"}
	for _, action := range []string{"approve", "reject"} {
		handler.PostPathJSON(business, "/applications/:id/"+action, services.Application.ReviewApplicationAction(action)).Name(".入驻申请." + applicationActionNames[action])
	}

	handler.GetList[bizquery.MerchantQuery, bizmodel.Merchant](business, "/merchants").Name(".商户.查看列表")
	business.Put("/merchants/status", handler.UpdateOnly[bizmodel.MerchantStatusUpdate, uint]).Name(".商户.更新状态")

	handler.GetList[bizquery.SysOrderQuery, bizmodel.Preorder](business, "/orders").Name(".订单.查看列表")
	handler.PostPath(business, "/orders/:id/cancel", services.Preorder.SysCancel).Name(".订单.取消")
	handler.PostPath(business, "/orders/:id/refund", services.Preorder.SysRefund).Name(".订单.退款")
	handler.GetList[bizquery.FeedbackQuery, bizmodel.Feedback](business, "/feedback").Name(".反馈.查看列表")
	handler.PutPathJSON(business, "/feedback/:id", services.Feedback.UpdateSys).Name(".反馈.处理")
	handler.GetList[bizquery.ActiveStallSessionQuery, bizmodel.StallSession](business, "/stalls/active").Name(".出摊.活跃列表")
}
