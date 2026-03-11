// main.go — Application lifecycle: startup, serve, graceful shutdown
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/job"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
)

func main() {
	cfg := config.Load()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	db, err := repository.NewDB(cfg.DB)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect db")
	}
	defer db.Close()

	d, err := wireDeps(db, cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to wire dependencies")
	}
	defer d.pgAuditLog.Close()

	allowedOrigins := []string{cfg.Server.AllowedOrigin, "http://localhost:3000", "http://localhost:3001", "http://localhost:5173", "http://localhost:8000"}
	router := registerRoutes(d.handlers, d.authService, allowedOrigins, cfg.Upload, logger)

	server := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		logger.Info().Msg("server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("server failed")
		}
	}()

	donationJob := job.NewDonationSnapshotJob(d.donationRepo, logger)
	donationJob.Start()
	sessionJob := job.NewSessionCleanupJob(d.sessionRepo, d.passwordResetRepo, d.notificationRepo, logger)
	sessionJob.Start()
	emailWorker := job.NewEmailWorker(d.emailQueue, d.emailService, logger)
	emailWorker.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	donationJob.Stop()
	sessionJob.Stop()
	emailWorker.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("forced shutdown")
	}
	logger.Info().Msg("server stopped")
}
