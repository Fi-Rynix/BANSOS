package middleware

import (
	"github.com/gofiber/fiber/v2"

	"bansos-backend/app/model"
	"bansos-backend/helper"
)

type PermissionSet interface {
	Can(role, permission string) bool
}

type StaticPermissionSet struct {
	grants map[string]map[string]struct{}
}

func NewStaticPermissionSet(grants map[string][]string) *StaticPermissionSet {
	set := &StaticPermissionSet{grants: make(map[string]map[string]struct{}, len(grants))}
	for role, perms := range grants {
		allowed := make(map[string]struct{}, len(perms))
		for _, p := range perms {
			allowed[p] = struct{}{}
		}
		set.grants[role] = allowed
	}
	return set
}

func (s *StaticPermissionSet) Can(role, permission string) bool {
	allowed, ok := s.grants[role]
	if !ok {
		return false
	}
	_, granted := allowed[permission]
	return granted
}

func RequirePermission(perms PermissionSet, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.CurrentUser(c)
		if !ok {
			return helper.Unauthorized("belum terautentikasi")
		}

		if perms == nil || !perms.Can(user.Role, permission) {
			return helper.Forbidden("role " + user.Role + " tidak memiliki hak " + permission)
		}
		return c.Next()
	}
}

func RequireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		user, ok := helper.CurrentUser(c)
		if !ok {
			return helper.Unauthorized("belum terautentikasi")
		}

		if _, granted := allowed[user.Role]; !granted {
			return helper.Forbidden("role Anda tidak berhak mengakses endpoint ini")
		}
		return c.Next()
	}
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

var DefaultPermissions = map[string][]string{
	model.RoleAdmin: {
		"module:list", "module:manage",
		"batch:list", "batch:manage",
		"claim:create", "claim:read:own", "claim:cancel",
		"claim:read:any", "claim:verify",
		"progress:read:own", "progress:write:own", "progress:read:any",
		"certificate:read:own", "certificate:issue", "certificate:verify",
		"content:manage", "user:manage",
	},
	model.RolePengelola: {
		"module:list", "module:manage",
		"batch:list",
		"claim:read:any", "claim:verify",
		"progress:read:any",
		"certificate:issue", "certificate:verify",
		"content:manage",
	},
	model.RolePeserta: {
		"module:list",
		"batch:list",
		"claim:create", "claim:read:own", "claim:cancel",
		"progress:read:own", "progress:write:own",
		"certificate:read:own", "certificate:verify",
	},
}

func NewDefaultPermissionSet() *StaticPermissionSet {
	return NewStaticPermissionSet(DefaultPermissions)
}
