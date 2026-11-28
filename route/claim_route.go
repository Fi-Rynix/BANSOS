package route

import (
	"github.com/gofiber/fiber/v2"

	"bansos-backend/app/service"
	"bansos-backend/helper"
	"bansos-backend/middleware"
)

func RegisterClaimRoutes(api fiber.Router, svc *service.ClaimService, perms middleware.PermissionSet) {
	readOwnOrAny := requireAnyPermission(perms, "claim:read:own", "claim:read:any")

	api.Get("/me/claims", middleware.RequireAuth, middleware.RequirePermission(perms, "claim:read:own"), svc.MyHistory)
	api.Get("/batches/:id/claims", middleware.RequireAuth, middleware.RequirePermission(perms, "claim:read:any"), svc.ListByBatch)

	claims := api.Group("/claims", middleware.RequireAuth)
	claims.Post("/", middleware.RequirePermission(perms, "claim:create"), middleware.RequireJSON, svc.Create)
	claims.Get("/:id", readOwnOrAny, svc.Get)
	claims.Get("/:id/logs", readOwnOrAny, svc.Logs)
	claims.Post("/:id/cancel", middleware.RequirePermission(perms, "claim:cancel"), svc.Cancel)
	claims.Patch("/:id/status", middleware.RequirePermission(perms, "claim:verify"), middleware.RequireJSON, svc.Verify)
}

// requireAnyPermission lolos kalau role punya salah satu permission. Kepemilikan
// data tetap dicek di service.
func requireAnyPermission(perms middleware.PermissionSet, permissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.CurrentUser(c)
		if !ok {
			return helper.Unauthorized("belum terautentikasi")
		}
		for _, p := range permissions {
			if perms != nil && perms.Can(user.Role, p) {
				return c.Next()
			}
		}
		return helper.Forbidden("role " + user.Role + " tidak memiliki hak membaca klaim")
	}
}
