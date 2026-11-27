package model

import "time"

const (
	ClaimPending    = "pending"
	ClaimApproved   = "approved"
	ClaimRejected   = "rejected"
	ClaimCancelled  = "cancelled"
	ClaimExpired    = "expired"
	ClaimWaitlisted = "waitlisted"
)

type Claim struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	BatchID   int        `json:"batch_id"`
	Status    string     `json:"status"`
	ClaimedAt time.Time  `json:"claimed_at"`
	DecidedAt *time.Time `json:"decided_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type ClaimDetail struct {
	Claim
	ModuleID      int       `json:"module_id"`
	ModuleCode    string    `json:"module_code"`
	ModuleTitle   string    `json:"module_title"`
	BatchOpensAt  time.Time `json:"batch_opens_at"`
	BatchClosesAt time.Time `json:"batch_closes_at"`
}

type ClaimLog struct {
	ID         int       `json:"id"`
	ClaimID    int       `json:"claim_id"`
	ActorID    int       `json:"actor_id"`
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

type ClaimListFilter struct {
	UserID  int
	BatchID int
	Status  string
	Limit   int
	Cursor  *Cursor
}

type CreateClaimRequest struct {
	BatchID int `json:"batch_id" validate:"required,min=1"`
}

type CancelClaimRequest struct {
	Note string `json:"note" validate:"max=255"`
}

type ClaimHistoryQuery struct {
	Status string `query:"status" validate:"omitempty,oneof=pending approved rejected cancelled expired waitlisted"`
	Limit  int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor string `query:"cursor"`
}

type VerifyClaimRequest struct {
	Status string `json:"status" validate:"required,oneof=approved rejected waitlisted"`
	Note   string `json:"note" validate:"max=255"`
}
