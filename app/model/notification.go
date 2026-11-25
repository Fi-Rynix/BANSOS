package model

import (
	"encoding/json"
	"time"
)

const (
	EventClaimApproved       = "claim.approved"
	EventCertificateIssued   = "certificate.issued"
	EventCertificateRevoked  = "certificate.revoked"
	TaskSendNotification     = "notification:send"
	QueueNotifications       = "notifications"
)

type OutboxEvent struct {
	ID        int64
	EventType string
	Payload   []byte
	CreatedAt time.Time
}

type NotificationJob struct {
	OutboxID  int64           `json:"outbox_id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

type Notification struct {
	ID        int64     `json:"id"`
	OutboxID  int64     `json:"outbox_id"`
	UserID    int       `json:"user_id"`
	Channel   string    `json:"channel"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	SentAt    time.Time `json:"sent_at"`
}

type Message struct {
	To      string
	Subject string
	Body    string
}

type ClaimDecisionEvent struct {
	UserID    int    `json:"user_id"`
	ClaimID   int    `json:"claim_id"`
	ModuleName string `json:"module_name"`
	Status    string `json:"status"`
}

type CertificateEvent struct {
	UserID     int    `json:"user_id"`
	Serial     string `json:"serial"`
	ModuleName string `json:"module_name"`
}