package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"bansos-backend/helper"
)

func Register(app *fiber.App, logger *slog.Logger, allowedOrigins string) {
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(corsPolicy(allowedOrigins))
	app.Use(RequestLogger(logger))
}

func corsPolicy(allowedOrigins string) fiber.Handler {
	if strings.TrimSpace(allowedOrigins) == "" {
		allowedOrigins = "http://localhost:5173"
	}
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
		ExposeHeaders: "Location",
	})
}

func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		requestID := helper.RequestID(c)
		status := c.Response().StatusCode()
		attrs := []any{
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("latency", time.Since(start)),
			slog.String("ip", c.IP()),
		}

		if status >= fiber.StatusInternalServerError {
			logger.Error("request_completed", attrs...)
		} else {
			logger.Info("request_completed", attrs...)
		}

		return err
	}
}

func RequireJSON(c *fiber.Ctx) error {
	if c.Method() == fiber.MethodGet || c.Method() == fiber.MethodDelete {
		return c.Next()
	}

	contentType := c.Get(fiber.HeaderContentType)
	if !strings.HasPrefix(strings.ToLower(contentType), fiber.MIMEApplicationJSON) {
		return helper.UnsupportedMediaType("Content-Type harus application/json")
	}
	return c.Next()
}
