package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"bansos-backend/config"
	"bansos-backend/database"
	"bansos-backend/route"
)

func startAPI(logger *slog.Logger, pool *pgxpool.Pool) *fiber.App {
	mongoClient, err := database.NewMongoClient(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke MongoDB", slog.String("error", err.Error()))
		return nil
	}

	app := config.NewApp(logger, route.Dependencies{
		Pool:  pool,
		Mongo: mongoClient,
	})

	app.Hooks().OnShutdown(func() error {
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
