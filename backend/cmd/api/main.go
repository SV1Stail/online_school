package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/SV1Stail/online_school/internal/auth"
	"github.com/SV1Stail/online_school/internal/conference"
	"github.com/SV1Stail/online_school/internal/config"
	"github.com/SV1Stail/online_school/internal/httpapi"
	"github.com/SV1Stail/online_school/internal/logging"
	"github.com/SV1Stail/online_school/internal/platform/postgres"
	redisstore "github.com/SV1Stail/online_school/internal/platform/redis"
	"github.com/SV1Stail/online_school/internal/schedule"
	"github.com/SV1Stail/online_school/internal/user"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := logging.New(cfg.LogLevel, cfg.LogPretty)
	logger.Info().
		Str("event", "service_starting").
		Str("environment", cfg.Environment).
		Str("address", cfg.HTTPAddress).
		Msg("service is starting")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	database, err := postgres.Open(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("database initialization failed")
	}
	defer database.Close()
	go cleanExpiredSessions(ctx, database, logger)

	cache := redisstore.New(
		cfg.RedisAddress,
		cfg.RedisPassword,
		cfg.RedisDB,
		cfg.LoginAttempts,
		cfg.LoginWindow,
	)
	if err := cache.Ping(ctx); err != nil {
		logger.Fatal().Err(err).Msg("redis initialization failed")
	}
	defer func() {
		if err := cache.Close(); err != nil {
			logger.Error().Err(err).Msg("redis close failed")
		}
	}()

	authService := auth.NewService(database, database, cache, cfg.SessionTTL)
	userService := user.NewService(database, database, database)
	scheduleService := schedule.NewService(database, database)
	conferenceService := conference.NewService(database, database)
	api, err := httpapi.New(
		cfg,
		authService,
		userService,
		scheduleService,
		conferenceService,
		database,
		cache,
		logger,
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("API initialization failed")
	}
	handler, err := api.Handler(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("HTTP handler initialization failed")
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info().Str("event", "service_started").Str("address", cfg.HTTPAddress).Msg("service started")
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info().Str("event", "service_stopping").Msg("shutdown signal received")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error().Err(err).Msg("HTTP server stopped unexpectedly")
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("graceful shutdown failed")
		_ = server.Close()
	}
	logger.Info().Str("event", "service_stopped").Msg("service stopped")
}

type sessionCleaner interface {
	DeleteExpiredSessions(ctx context.Context) error
}

func cleanExpiredSessions(ctx context.Context, cleaner sessionCleaner, logger zerolog.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Error().Msg("ctx done")

			return
		case <-ticker.C:
			func() {
				cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				err := cleaner.DeleteExpiredSessions(cleanupCtx)
				if err != nil && !errors.Is(err, context.Canceled) {
					logger.Error().Err(err).Msg("expired session cleanup failed")
				}
			}()
		}
	}
}
