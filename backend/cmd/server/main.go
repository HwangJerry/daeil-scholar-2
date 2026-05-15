// main.go — Application lifecycle: startup, serve, graceful shutdown
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/job"
	"github.com/dflh-saf/backend/internal/observability"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	debugHook := observability.NewHook(cfg.DebugAgent)
	if debugHook != nil {
		logger = logger.Hook(debugHook)
		log.Logger = log.Logger.Hook(debugHook)
		logger.Info().
			Str("project", cfg.DebugAgent.Project).
			Str("environment", cfg.DebugAgent.Environment).
			Msg("debug agent reporter enabled")
	}

	db, err := repository.NewDB(cfg.DB)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect db")
	}
	defer db.Close()

	d, err := wireDeps(db, cfg, logger, debugHook)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to wire dependencies")
	}
	defer d.pgAuditLog.Close()

	rawOrigins := strings.Split(cfg.Server.AllowedOrigin, ",")
	for i := range rawOrigins {
		rawOrigins[i] = strings.TrimSpace(rawOrigins[i])
	}
	allowedOrigins := rawOrigins
	if cfg.Environment == "dev" {
		allowedOrigins = append(allowedOrigins,
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:5173",
			"http://localhost:8000",
			"http://127.0.0.1:3001",
			"http://127.0.0.1:5173",
			"http://127.0.0.1:8000",
			"https://client-macbook.tail04b57d.ts.net",
		)
	}
	router := registerRoutes(d.handlers, d.authService, d.cacheStore, allowedOrigins, cfg, logger, debugHook)

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

	// DISABLED 2026-04-28: external donation redirect (dangled — see /home/jerryhwang/.claude/plans/drifting-gliding-hopcroft.md).
	// donationJob and subscriptionBillingJob are wired in wireDeps but intentionally not Start()ed
	// while donations are routed to the external Happy Nanum platform. Re-enable by restoring the
	// Start()/Stop() pairs below.
	// donationJob := d.donationJob
	// donationJob.Start()
	sessionJob := job.NewSessionCleanupJob(d.sessionRepo, d.passwordResetRepo, logger)
	sessionJob.Start()
	emailWorker := job.NewEmailWorker(d.emailQueue, d.emailService, logger)
	emailWorker.Start()
	// subscriptionBillingJob := d.subscriptionBillingJob
	// subscriptionBillingJob.Start()
	visitJob := d.visitJob
	visitJob.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// donationJob.Stop()
	sessionJob.Stop()
	emailWorker.Stop()
	// subscriptionBillingJob.Stop()
	visitJob.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("forced shutdown")
	}
	logger.Info().Msg("server stopped")
}
