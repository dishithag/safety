package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.crwd.dev/ce/zerotrust-analytics/internal/analyticsapi"
	"go.crwd.dev/ce/zerotrust-analytics/internal/analyticsapi/domain"
	"go.crwd.dev/ce/zerotrust-analytics/internal/analyticsapi/gateways/api"
	"go.crwd.dev/ce/zerotrust-analytics/internal/analyticsapi/gateways/repos"
	"go.crwd.dev/ce/zerotrust-analytics/internal/analyticsapi/usecase/narrative"
)

const serviceName = "cs.analyticsapi"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", serviceName)

	cfg := domain.LoadConfig(serviceName)

	narrativeStore, err := repos.NewNarrativeStore(cfg)
	if err != nil {
		logger.Error("failed to initialize narrative store", "err", err)
		os.Exit(1)
	}

	narrativeSvc := narrative.NewService(logger, narrativeStore)
	manager := api.NewManager(logger, narrativeSvc)
	service := analyticsapi.NewService(logger, cfg, manager)

	// Shut down cleanly on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- service.Start() }()

	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := service.Stop(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "err", err)
			os.Exit(1)
		}
	}
}
