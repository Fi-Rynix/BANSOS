package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"bansos-backend/app/model"
	"bansos-backend/app/repository"
	"bansos-backend/helper"
)

type UserContactFetcher interface {
	GetContact(ctx context.Context, userID int) (name, email string, err error)
}

type NotificationWorker struct {
	users         UserContactFetcher
	notifications repository.NotificationRepository
	tx            *repository.TxManager
	mailer        helper.Mailer
	logger        *slog.Logger
}

func NewNotificationWorker(
	users UserContactFetcher,
	notifications repository.NotificationRepository,
	tx *repository.TxManager,
	mailer helper.Mailer,
	logger *slog.Logger,
) *NotificationWorker {
	return &NotificationWorker{
		users: users, notifications: notifications,
		tx: tx, mailer: mailer, logger: logger,
	}
}

func (w *NotificationWorker) HandleSend(ctx context.Context, t *asynq.Task) error {
	var job model.NotificationJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("%w: payload task rusak: %v", asynq.SkipRetry, err)
	}
	
	if w.users == nil {
		return fmt.Errorf("%w: UserContactFetcher belum dikonfigurasi", asynq.SkipRetry)
	}

	var userID int
	var msg model.Message

	switch job.EventType {
	case model.EventClaimApproved:
		var event model.ClaimDecisionEvent
		if err := json.Unmarshal(job.Payload, &event); err != nil {
			return fmt.Errorf("%w: payload claim rusak: %v", asynq.SkipRetry, err)
		}
		userID = event.UserID
		name, email, err := w.users.GetContact(ctx, userID)
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("%w: user %d tidak ditemukan", asynq.SkipRetry, userID)
		}
		if err != nil {
			return err
		}
		msg = model.Message{
			To:      email,
			Subject: "Klaim BANSOS Disetujui",
			Body:    fmt.Sprintf("Halo %s, klaim Anda untuk modul %s telah disetujui.", name, event.ModuleName),
		}

	case model.EventCertificateIssued:
		var event model.CertificateEvent
		if err := json.Unmarshal(job.Payload, &event); err != nil {
			return fmt.Errorf("%w: payload certificate rusak: %v", asynq.SkipRetry, err)
		}
		userID = event.UserID
		name, email, err := w.users.GetContact(ctx, userID)
		if errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("%w: user %d tidak ditemukan", asynq.SkipRetry, userID)
		}
		if err != nil {
			return err
		}
		msg = model.Message{
			To:      email,
			Subject: "Sertifikat BANSOS Terbit",
			Body:    fmt.Sprintf("Selamat %s! Sertifikat untuk modul %s telah terbit dengan serial %s.", name, event.ModuleName, event.Serial),
		}

	default:
		return fmt.Errorf("%w: event_type tidak dikenal: %q", asynq.SkipRetry, job.EventType)
	}

	alreadySent := false
	err := w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		notifRepo := w.notifications.WithTx(tx)
		claimed, err := notifRepo.Claim(ctx, model.Notification{
			OutboxID:  job.OutboxID,
			UserID:    userID,
			Channel:   "email",
			Subject:   msg.Subject,
			Body:      msg.Body,
		})
		if err != nil {
			return err
		}
		if !claimed {
			alreadySent = true
			return nil
		}
		return w.mailer.Send(ctx, msg)
	})
	if errors.Is(err, helper.ErrPermanentDelivery) {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	if err != nil {
		return err
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("notification_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event", job.EventType),
		slog.Int("user_id", userID),
		slog.Int("retry", retry),
		slog.Bool("already_sent", alreadySent),
	)
	return nil
}