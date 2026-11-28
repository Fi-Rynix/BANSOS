package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"bansos-backend/app/repository"
	"bansos-backend/app/service"
	"bansos-backend/config"
	"bansos-backend/database"
	"bansos-backend/helper"
	"bansos-backend/middleware"
	"bansos-backend/route"
)

func startAPI(logger *slog.Logger, pool *pgxpool.Pool) *fiber.App {
	mongoClient, err := database.NewMongoClient(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke MongoDB", slog.String("error", err.Error()))
		return nil
	}

	jwtManager := helper.NewJWTManager(
		config.GetEnv("JWT_SECRET", ""),
		config.GetEnv("JWT_ISSUER", "bansos"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)
	middleware.SetJWTManager(jwtManager)
	middleware.SetAllowStubToken(config.GetEnv("ALLOW_STUB_TOKEN", "false") == "true")

	roleRepo := repository.NewRoleRepository()
	permissionSet := newPermissionSet(logger, pool, roleRepo)
	stopReload := startPermissionReload(logger, permissionSet)

	userRepo := repository.NewUserRepository()
	tokenRepo := repository.NewTokenRepository()
	refreshTTL := time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_DAYS", 7)) * 24 * time.Hour

	app := config.NewApp(logger, route.Dependencies{
		Pool:        pool,
		Mongo:       mongoClient,
		AuthService: service.NewAuthService(pool, userRepo, tokenRepo, jwtManager, refreshTTL),
		UserService: service.NewUserService(pool, userRepo, tokenRepo),
		Permissions: permissionSet,
	})

	app.Hooks().OnShutdown(func() error {
		close(stopReload)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return mongoClient.Disconnect(ctx)
	})

	port := config.GetEnv("APP_PORT", "3000")

	go func() {
		logger.Info("HTTP berjalan", slog.String("port", port))
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server berhenti", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	return app
}

// newPermissionSet membaca role_permissions dari database. Kalau pembacaan
// pertama gagal, dipakai peta bawaan supaya aplikasi tetap bisa menyala dan
// masalahnya terlihat di log.
func newPermissionSet(logger *slog.Logger, pool *pgxpool.Pool, roles repository.RoleRepository) *middleware.DatabasePermissionSet {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	grants, err := roles.LoadPermissions(ctx, pool)
	if err != nil {
		logger.Error("gagal membaca permission dari database, memakai bawaan",
			slog.String("error", err.Error()))
		grants = middleware.DefaultPermissions
	}
	return middleware.NewDatabasePermissionSet(func() (map[string][]string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return roles.LoadPermissions(ctx, pool)
	}, grants)
}

func startPermissionReload(logger *slog.Logger, set *middleware.DatabasePermissionSet) chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if err := set.Reload(); err != nil {
					logger.Warn("gagal menyegarkan permission",
						slog.String("error", err.Error()))
				}
			}
		}
	}()

	return stop
}
