package model

import "time"

const (
	BatchDraft     = "draft"
	BatchScheduled = "scheduled"
	BatchOpen      = "open"
	BatchClosed    = "closed"
)

type Batch struct {
	ID        int       `json:"id"`
	ModuleID  int       `json:"module_id"`
	Quota     int       `json:"quota"`
	Allocated int       `json:"allocated"`
	Remaining int       `json:"remaining"`
	OpensAt   time.Time `json:"opens_at"`
	ClosesAt  time.Time `json:"closes_at"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateBatchRequest struct {
	Quota    int       `json:"quota" validate:"required,min=1,max=100000"`
	OpensAt  time.Time `json:"opens_at" validate:"required"`
	ClosesAt time.Time `json:"closes_at" validate:"required"`
}

type UpdateBatchRequest struct {
	Quota    int       `json:"quota" validate:"required,min=1,max=100000"`
	OpensAt  time.Time `json:"opens_at" validate:"required"`
	ClosesAt time.Time `json:"closes_at" validate:"required"`
}

type ChangeBatchStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=draft scheduled open closed"`
}
