package middleware

import (
	"sync"

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

var _ PermissionSet = (*StaticPermissionSet)(nil)
var _ PermissionSet = (*DatabasePermissionSet)(nil)

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

type PermissionLoader func() (map[string][]string, error)

// DatabasePermissionSet membaca role_permissions dari database sekali, lalu
// menyimpannya di memori. Bentuk interface PermissionSet sengaja tidak berubah
// supaya route milik anggota lain tidak perlu disentuh.
type DatabasePermissionSet struct {
	mu        sync.RWMutex
	grants    map[string]map[string]struct{}
	loader    PermissionLoader
	lastError error
}

func NewDatabasePermissionSet(loader PermissionLoader, initial map[string][]string) *DatabasePermissionSet {
	set := &DatabasePermissionSet{loader: loader}
	set.replace(initial)
	return set
}

func (s *DatabasePermissionSet) replace(grants map[string][]string) {
	byRole := make(map[string]map[string]struct{}, len(grants))
	for role, permissions := range grants {
		allowed := make(map[string]struct{}, len(permissions))
		for _, permission := range permissions {
			allowed[permission] = struct{}{}
		}
		byRole[role] = allowed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.grants = byRole
}

func (s *DatabasePermissionSet) Can(role, permission string) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	allowed, ok := s.grants[role]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	_, granted := allowed[permission]
	return granted
}

// Reload menarik ulang peta permission dari database. Saat gagal, peta lama
// dipertahankan supaya permintaan yang sedang berjalan tidak ikut gagal.
func (s *DatabasePermissionSet) Reload() error {
	grants, err := s.loader()
	if err != nil {
		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()
		return err
	}

	s.replace(grants)
	s.mu.Lock()
	s.lastError = nil
	s.mu.Unlock()
	return nil
}

func (s *DatabasePermissionSet) LastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
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

// DefaultPermissions adalah cadangan kalau role_permissions belum terbaca dari
// database. Isi database tetap jadi sumber utama lewat DatabasePermissionSet.
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
