package model

import "time"

const (
	ModuleLevelBeginner     = "beginner"
	ModuleLevelIntermediate = "intermediate"
	ModuleLevelAdvanced     = "advanced"
)

const (
	ModuleStatusActive   = "active"
	ModuleStatusInactive = "inactive"
	ModuleStatusAll      = "all"
)

type Module struct {
	ID          int       `json:"id"`
	Code        string    `json:"code"`
	Title       string    `json:"title"`
	Category    string    `json:"category"`
	Level       string    `json:"level"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type ModuleListFilter struct {
	Category string
	Level    string
	Status   string
	Limit    int
	Cursor   *Cursor
}

type CreateModuleRequest struct {
	Code        string `json:"code" validate:"required,min=2,max=30,modulecode"`
	Title       string `json:"title" validate:"required,min=3,max=150"`
	Category    string `json:"category" validate:"required,min=2,max=50"`
	Level       string `json:"level" validate:"required,oneof=beginner intermediate advanced"`
	Description string `json:"description" validate:"max=2000"`
}

type UpdateModuleRequest struct {
	Code        string `json:"code" validate:"required,min=2,max=30,modulecode"`
	Title       string `json:"title" validate:"required,min=3,max=150"`
	Category    string `json:"category" validate:"required,min=2,max=50"`
	Level       string `json:"level" validate:"required,oneof=beginner intermediate advanced"`
	Description string `json:"description" validate:"max=2000"`
	IsActive    *bool  `json:"is_active" validate:"required"`
}

type ModuleListQuery struct {
	Category string `query:"category" validate:"omitempty,max=50"`
	Level    string `query:"level" validate:"omitempty,oneof=beginner intermediate advanced"`
	Status   string `query:"status" validate:"omitempty,oneof=active inactive all"`
	Limit    int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor   string `query:"cursor"`
}
