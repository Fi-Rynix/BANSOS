package helper

import (
	"github.com/gofiber/fiber/v2"

	"bansos-backend/app/model"
)

const (
	LocalsAuthUser  = "authUser"
	LocalsRequestID = "requestid"
)

func CurrentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	user, ok := c.Locals(LocalsAuthUser).(model.AuthUser)
	return user, ok
}

func RequestID(c *fiber.Ctx) string {
	rid, _ := c.Locals(LocalsRequestID).(string)
	return rid
}
