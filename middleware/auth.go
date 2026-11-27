package middleware

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"bansos-backend/helper"
)

const (
	HeaderAuthScheme = "Bearer"
	HeaderAuthRealm  = `Bearer realm="bansos"`
)

var (
	ErrMissingToken = errors.New("header Authorization kosong")
	ErrMalformed    = errors.New("header Authorization bukan skema Bearer")
	ErrStubToken    = errors.New("verifikasi token belum diimplementasikan")
)

func RequireAuth(c *fiber.Ctx) error {
	token, err := bearerToken(c)
	if err != nil {
		c.Set(fiber.HeaderWWWAuthenticate, HeaderAuthRealm)
		return helper.Unauthorized("header Authorization tidak ada atau salah bentuk")
	}

	authUser, err := parseStubToken(token)
	if err != nil {
		c.Set(fiber.HeaderWWWAuthenticate, HeaderAuthRealm)
		if errors.Is(err, ErrStubToken) {
			return helper.Unauthorized("verifikasi token belum aktif, hubungi Git Master")
		}
		return helper.Unauthorized("access token tidak valid")
	}

	c.Locals(helper.LocalsAuthUser, authUser)
	return c.Next()
}

func RequireOptionalAuth(c *fiber.Ctx) error {
	token, err := bearerToken(c)
	if err != nil {
		return c.Next()
	}

	authUser, err := parseStubToken(token)
	if err != nil {
		return c.Next()
	}

	c.Locals(helper.LocalsAuthUser, authUser)
	return c.Next()
}

func bearerToken(c *fiber.Ctx) (string, error) {
	header := c.Get(fiber.HeaderAuthorization)
	if header == "" {
		return "", ErrMissingToken
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], HeaderAuthScheme) {
		return "", ErrMalformed
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", ErrMissingToken
	}
	return token, nil
}
