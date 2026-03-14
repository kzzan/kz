package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example/internal/consumer"
	"example/internal/cron"
	"example/internal/server"
	"example/pkg"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

func main() {
	injector := do.New(
		pkg.BasePackage,
		server.Package,
	)

	logger := do.MustInvoke[*zerolog.Logger](injector)
	srv    := do.MustInvoke[*server.Server](injector)

	logger.Info().Msg("Starting application")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if manager, err := do.Invoke[*consumer.Manager](injector); err == nil {
		manager.Start(ctx)
		logger.Info().Msg("Consumer manager started")
	}

	if scheduler, err := do.Invoke[*cron.Scheduler](injector); err == nil {
		scheduler.Start()
		logger.Info().Msg("Cron scheduler started")
		defer scheduler.Stop()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- srv.Start()
	}()

	select {
	case err := <-serverErrors:
		logger.Error().Err(err).Msg("Server exited with error")
	case sig := <-sigChan:
		logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("Error during graceful shutdown")
		}
	}

	if _, err := injector.ShutdownOnSignals(); err != nil {
		logger.Error().Err(err).Msg("Error shutting down injector")
	}
}