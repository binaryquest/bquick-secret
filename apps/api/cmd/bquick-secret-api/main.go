package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bquick-secret/apps/api/internal/config"
	"bquick-secret/apps/api/internal/email"
	"bquick-secret/apps/api/internal/httpapi"
	"bquick-secret/apps/api/internal/store"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("startup_failed", "category", "database")
		os.Exit(1)
	}
	defer db.Close()

	mailer, err := email.NewSES(ctx, cfg.SESRegion, cfg.SESFromEmail)
	if err != nil {
		logger.Warn("email_disabled", "category", "ses_config")
		mailer = email.Disabled{}
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.New(cfg, db, mailer, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go runCleanup(ctx, db, logger)

	go func() {
		logger.Info("api_started", "status", 200)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api_stopped", "category", "listen")
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown_failed", "category", "http")
	}
}

func runCleanup(ctx context.Context, db *store.Store, logger *slog.Logger) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		expired, purged, err := db.CleanupExpired(ctx, time.Now().UTC())
		if err != nil {
			logger.Warn("cleanup_failed", "category", "database")
		} else if expired > 0 || purged > 0 {
			logger.Info("cleanup_completed", "status", 200)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
