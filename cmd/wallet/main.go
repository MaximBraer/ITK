package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ITK/internal/api"
	"ITK/internal/api/handlers"
	"ITK/internal/config"
	"ITK/internal/repository"
	"ITK/internal/service"
	"ITK/pkg/postgres"
)

// @title Wallet Service API
// @version 1.0
// @description REST API for managing wallets with deposit and withdrawal operations
// @host localhost:8080
// @BasePath /
func main() {
	cfg := config.MustLoad()

	logger := setupLogger(cfg.Env)
	logger.Info("starting wallet service", slog.String("env", cfg.Env))

	pool, err := postgres.NewPool(context.Background(), cfg.DB, logger)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	walletRepo := repository.New(pool, logger, repository.Config{
		MaxRetries:  cfg.Retry.MaxAttempts,
		BaseDelayMS: cfg.Retry.BaseDelayMS,
	})
	walletService := service.New(walletRepo, logger)
	walletHandler := handlers.New(walletService, logger)

	router := api.NewRouter(logger, walletHandler)

	server := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		logger.Info("starting HTTP server", slog.String("address", cfg.HTTPServer.Address))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	logger.Info("wallet service started successfully")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
	}

	logger.Info("server exited gracefully")
}

func setupLogger(env string) *slog.Logger {
	var handler slog.Handler

	if env == "local" || env == "development" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	return slog.New(handler)
}
