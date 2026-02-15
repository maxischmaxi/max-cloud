package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/max-cloud/api/internal/config"
	"github.com/max-cloud/api/internal/email"
	"github.com/max-cloud/api/internal/orchestrator"
	"github.com/max-cloud/api/internal/reconciler"
	"github.com/max-cloud/api/internal/server"
	"github.com/max-cloud/api/internal/store"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	var st store.ServiceStore
	var authSt store.AuthStore

	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		pg, err := store.NewPostgres(ctx, cfg.DatabaseURL)
		if err != nil {
			logger.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer pg.Close()
		st = pg
		authSt = pg
		logger.Info("using PostgreSQL store")

		if cfg.DevMode && cfg.DevOrgUID != "" {
			if err := pg.EnsureDevOrg(context.Background(), cfg.DevOrgUID); err != nil {
				logger.Warn("failed to ensure dev org", "error", err)
			} else {
				logger.Info("dev org ensured", "org_id", cfg.DevOrgUID)
			}
		}
	} else {
		mem := store.NewMemory()
		st = mem
		authSt = mem
		logger.Info("using in-memory store (no DATABASE_URL set)")
	}

	var orch orchestrator.Orchestrator
	if cfg.KubeconfigPath != "" {
		k, err := orchestrator.NewKnative(logger, cfg.KubeconfigPath, cfg.KnativeNamespace, cfg.RegistryURL, cfg.RegistryJWTSecret)
		if err != nil {
			logger.Error("failed to create knative orchestrator", "error", err)
			os.Exit(1)
		}
		orch = k
		logger.Info("using Knative orchestrator", "namespace", cfg.KnativeNamespace)
	} else {
		orch = orchestrator.NewNoop(logger)
		logger.Info("using Noop orchestrator (no KUBECONFIG set)")
	}

	if cfg.ResendAPIKey == "" && !cfg.DevMode {
		logger.Error("RESEND_API_KEY is required")
		os.Exit(1)
	}
	emailSender := email.NewResend(cfg.ResendAPIKey, cfg.EmailFrom)
	logger.Info("using Resend email sender", "from", cfg.EmailFrom)

	srv := server.New(logger, st, authSt, orch, emailSender, cfg.InviteExpiration, cfg.DevMode, cfg.DevOrgUID, cfg.RegistryURL, cfg.RegistryJWTSecret, cfg.RegistryTokenExpiry)

	rec := reconciler.New(logger, st, orch, cfg.ReconcileInterval)
	reconcilerCtx, reconcilerCancel := context.WithCancel(context.Background())
	defer reconcilerCancel()
	go rec.Run(reconcilerCtx)
	logger.Info("reconciler started", "interval", cfg.ReconcileInterval)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting API server", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")
	reconcilerCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("forced shutdown", "error", err)
	}
	logger.Info("server stopped")
}
