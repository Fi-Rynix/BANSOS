package route

import (
	"github.com/gofiber/fiber/v2"

	"bansos-backend/app/service"
	"bansos-backend/middleware"
)

func RegisterContentRoutes(api fiber.Router, svc *service.ContentService, perms middleware.PermissionSet) {
	content := api.Group("/modules/:id/content", middleware.RequireAuth)

	content.Get("/", middleware.RequirePermission(perms, "module:list"), svc.Get)
	content.Put("/", middleware.RequirePermission(perms, "content:manage"), middleware.RequireJSON, svc.Upsert)
}