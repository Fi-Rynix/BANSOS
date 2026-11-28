package middleware

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gofiber/fiber/v2"

	"bansos-backend/app/model"
	"bansos-backend/helper"
)

const (
	HeaderAuthScheme = "Bearer"
	HeaderAuthRealm  = `Bearer realm="bansos"`
)

var (
	ErrMissingToken = errors.New("header Authorization kosong")
	ErrMalformed    = errors.New("header Authorization bukan skema Bearer")
)

// RequireAuth memverifikasi access token lalu menyimpan identitas pemanggil di
// locals. Stub token masih diterima selama ALLOW_STUB_TOKEN aktif, supaya uji
// beban war slot tidak perlu login 500 akun lebih dulu.
func RequireAuth(c *fiber.Ctx) error {
	authUser, err := currentAuthUser(c)
	if err != nil {
		c.Set(fiber.HeaderWWWAuthenticate, HeaderAuthRealm)
		return err
	}

	c.Locals(helper.LocalsAuthUser, authUser)
	return c.Next()
}

func RequireOptionalAuth(c *fiber.Ctx) error {
	authUser, err := currentAuthUser(c)
	if err != nil {
		return c.Next()
	}

	c.Locals(helper.LocalsAuthUser, authUser)
	return c.Next()
}

func currentAuthUser(c *fiber.Ctx) (model.AuthUser, error) {
	token, err := bearerToken(c)
	if err != nil {
		return model.AuthUser{}, helper.Unauthorized("header Authorization tidak ada atau salah bentuk")
	}

	if AllowStubToken() && strings.HasPrefix(token, stubTokenPrefix) {
		user, stubErr := parseStubToken(token)
		if stubErr != nil {
			return model.AuthUser{}, helper.Unauthorized("access token tidak valid")
		}
		return user, nil
	}

	user, err := Verifier().Parse(token)
	switch {
	case errors.Is(err, helper.ErrExpiredToken):
		return model.AuthUser{}, helper.Unauthorized("access token kedaluwarsa")
	case err != nil:
		return model.AuthUser{}, helper.Unauthorized("access token tidak valid")
	}
	return user, nil
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

var (
	managerMu      sync.RWMutex
	manager        *helper.JWTManager
	allowStubToken atomic.Bool
)

// SetJWTManager dipasang sekali saat aplikasi dinyalakan dari main.
func SetJWTManager(m *helper.JWTManager) {
	managerMu.Lock()
	defer managerMu.Unlock()
	manager = m
}

func Verifier() *helper.JWTManager {
	managerMu.RLock()
	defer managerMu.RUnlock()
	return manager
}

// AllowStubToken hanya menyala kalau config mengizinkannya, dan config bawaan
// menolak stub. Uji beban menyalakannya lewat env di lingkungan uji.
func AllowStubToken() bool {
	return allowStubToken.Load()
}

func SetAllowStubToken(v bool) {
	allowStubToken.Store(v)
}

func RequireSelf(paramName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.CurrentUser(c)
		if !ok {
			return helper.Unauthorized("belum terautentikasi")
		}

		target, err := c.ParamsInt(paramName)
		if err != nil || target < 1 {
			return helper.BadRequest("parameter " + paramName + " tidak valid")
		}
		if user.UserID != target {
			return helper.Forbidden("hanya boleh mengakses data milik sendiri")
		}
		return c.Next()
	}
}
