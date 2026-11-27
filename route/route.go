package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"bansos-backend/app/repository"
	"bansos-backend/app/service"
	"bansos-backend/helper"
	"bansos-backend/middleware"
)

type Dependencies struct {
	Pool          *pgxpool.Pool
	Mongo         *mongo.Client
	EnableSwagger bool
}

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")
	api.Get("/health", healthCheck(deps.Pool, deps.Mongo))

	perms := middleware.NewDefaultPermissionSet()
	api.Get("/me", middleware.RequireAuth, currentUser)
	api.Get("/me/progress", middleware.RequireAuth, middleware.RequirePermission(perms, "progress:read:own"), currentUser)
	api.Get("/admin/users/:id", middleware.RequireAuth, middleware.RequirePermission(perms, "user:manage"), middleware.RequireSelf("id"), currentUser)

	moduleRepo := repository.NewModuleRepository()
	moduleService := service.NewModuleService(deps.Pool, moduleRepo, perms)
	RegisterModuleRoutes(api, moduleService, perms)

	batchRepo := repository.NewBatchRepository()
	batchService := service.NewBatchService(deps.Pool, batchRepo, moduleRepo, perms)
	RegisterBatchRoutes(api, batchService, perms)

	claimService := service.NewClaimService(deps.Pool, repository.NewClaimRepository(), batchRepo, perms)
	RegisterClaimRoutes(api, claimService, perms)
}

func currentUser(c *fiber.Ctx) error {
	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	return helper.Success(c, fiber.StatusOK, "OK", "identitas token", user)
}

func healthCheck(pool *pgxpool.Pool, mongoClient *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("PostgreSQL tidak dapat dihubungi")
		}
		if err := mongoClient.Ping(ctx, nil); err != nil {
			return helper.ServiceUnavailable("MongoDB tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "OK",
			"server, PostgreSQL, dan MongoDB berjalan", nil)
	}
}
