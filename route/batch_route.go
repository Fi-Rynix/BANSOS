package route

import (
	"github.com/gofiber/fiber/v2"

	"bansos-backend/app/service"
	"bansos-backend/middleware"
)

func RegisterBatchRoutes(api fiber.Router, svc *service.BatchService, perms middleware.PermissionSet) {
	list := middleware.RequirePermission(perms, "batch:list")
	manage := middleware.RequirePermission(perms, "batch:manage")

	api.Get("/modules/:id/batches", middleware.RequireAuth, list, svc.ListByModule)
	api.Post("/modules/:id/batches", middleware.RequireAuth, manage, middleware.RequireJSON, svc.Create)

	batches := api.Group("/batches", middleware.RequireAuth)
	batches.Get("/:id", list, svc.Get)
	batches.Put("/:id", manage, middleware.RequireJSON, svc.Update)
	batches.Patch("/:id/status", manage, middleware.RequireJSON, svc.ChangeStatus)
	batches.Delete("/:id", manage, svc.Delete)
}
