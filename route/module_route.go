package route

import (
	"github.com/gofiber/fiber/v2"

	"bansos-backend/app/service"
	"bansos-backend/middleware"
)

func RegisterModuleRoutes(api fiber.Router, svc *service.ModuleService, perms middleware.PermissionSet) {
	modules := api.Group("/modules", middleware.RequireAuth)

	modules.Get("/", middleware.RequirePermission(perms, "module:list"), svc.List)
	modules.Get("/:id", middleware.RequirePermission(perms, "module:list"), svc.Get)

	manage := middleware.RequirePermission(perms, "module:manage")
	modules.Post("/", manage, middleware.RequireJSON, svc.Create)
	modules.Put("/:id", manage, middleware.RequireJSON, svc.Update)
	modules.Delete("/:id", manage, svc.Delete)
}
