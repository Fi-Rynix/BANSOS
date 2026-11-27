package model

import "time"

const (
	RoleAdmin     = "admin"
	RolePengelola = "pengelola"
	RolePeserta   = "peserta"
)

func IsKnownRole(role string) bool {
	switch role {
	case RoleAdmin, RolePengelola, RolePeserta:
		return true
	}
	return false
}

type AuthUser struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type CursorMeta struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

type Cursor struct {
	CreatedAt time.Time
	ID        int
}
