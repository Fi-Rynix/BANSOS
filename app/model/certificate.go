package model

import (
	"time"
)

type Certificate struct {
	ID        int        `json:"id"`
	Serial    string     `json:"serial"`
	UserID    int        `json:"user_id"`
	ModuleID  int        `json:"module_id"`
	ClaimID   int        `json:"claim_id"`
	Status    string     `json:"status"`
	IssuedAt  *time.Time `json:"issued_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type CertificateIssueRequest struct {
	ClaimID int `json:"claim_id" validate:"required,min=1"`
}

type CertificateResponse struct {
	Serial   string `json:"serial"`
	UserName string `json:"user_name"`
	Module   string `json:"module"`
	Status   string `json:"status"`
	IssuedAt string `json:"issued_at"`
}