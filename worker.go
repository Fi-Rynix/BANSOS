package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"bansos-backend/app/model"
	"bansos-backend/app/repository"
	"bansos-backend/app/service"
	"bansos-backend/config"
	"bansos-backend/helper"
)

const (
	relayShutdownTimeout = 10 * time.Second
)

func startWorker(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool) func() {
	mailer, err := helper.NewLogMailer(
		"logs/mail.log",
		time.Duration(config.GetEnvInt("MAILER_DELAY_MS", 1500))*time.Millisecond,
		config.GetEnvInt("MAILER_FAIL_EVERY", 0),
	)
	if err != nil {
		logger.Error("gagal menyiapkan mailer", slog.String("error", err.Error()))
		return nil
	}

	outboxRepo := repository.NewOutboxRepository(pool)
	notifRepo := repository.NewNotificationRepository(pool)
	txManager := repository.NewTxManager(pool)

	client := asynq.NewClient(config.RedisOpt())

	relay := service.NewOutboxRelay(
		outboxRepo, txManager, client, logger,
		time.Duration(config.GetEnvInt("RELAY_INTERVAL_MS", 1000))*time.Millisecond,
		config.GetEnvInt("RELAY_BATCH", 100),
	)

	var users service.UserContactFetcher
	if users == nil {
		logger.Warn("UserContactFetcher nil — task notifikasi bisa panic saat dijalankan")
	}

	worker := service.NewNotificationWorker(
		users,
		notifRepo, txManager, mailer, logger,
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(model.TaskSendNotification, worker.HandleSend)

	srv := config.NewWorkerServer(logger)
	if err := srv.Start(mux); err != nil {
		logger.Error("gagal menyalakan worker", slog.String("error", err.Error()))
		_ = client.Close()
		return nil
	}

	workerCtx, cancelWorker := context.WithCancel(ctx)

	relayDone := make(chan struct{})
	go func() {
		relay.Run(workerCtx)
		close(relayDone)
	}()

	logger.Info("worker berjalan")

	return sync.OnceFunc(func() {
		cancelWorker()
		select {
		case <-relayDone:
			logger.Info("outbox relay berhenti")
		case <-time.After(relayShutdownTimeout):
			logger.Warn("outbox relay tidak berhenti tepat waktu",
				slog.Duration("timeout", relayShutdownTimeout))
		}

		srv.Shutdown()

		if err := client.Close(); err != nil {
			logger.Warn("gagal menutup asynq client",
				slog.String("error", err.Error()))
		}
	})
}