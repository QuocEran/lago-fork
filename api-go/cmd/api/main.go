package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/getlago/lago/api-go/config"
	"github.com/getlago/lago/api-go/config/database"
	cfgredis "github.com/getlago/lago/api-go/config/redis"
	"github.com/getlago/lago/api-go/internal/server"
	"github.com/getlago/lago/api-go/utils"
)

func main() {
	cfg := config.Load()

	logLevel := slog.LevelInfo
	if cfg.Env == "development" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(
		utils.NewLevelHandler(
			logLevel,
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}),
		),
	).With("service", "lago-api")
	slog.SetDefault(logger)

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      cfg.Env,
		AttachStacktrace: true,
	}); err != nil {
		slog.Warn("sentry init failed", slog.String("error", err.Error()))
	}
	defer sentry.Flush(2 * time.Second)

	db, err := database.NewConnection(database.Config{
		URL:      cfg.DatabaseURL,
		MaxConns: cfg.DatabaseMaxConns,
	})
	if err != nil {
		utils.LogAndPanic(err, "failed to connect to database")
	}
	defer db.Close()

	sqlDB, err := db.Connection.DB()
	if err != nil {
		utils.LogAndPanic(err, "failed to get sql.DB from gorm")
	}

	_, redisErr := cfgredis.NewRedisDB(context.Background(), cfgredis.Config{
		Address:  cfg.RedisURL,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		UseTLS:   cfg.RedisTLS,
	})
	if redisErr != nil {
		slog.Warn("failed to connect to redis", slog.String("error", redisErr.Error()))
	}

	version := utils.GetEnvOrDefault("APP_VERSION", "dev")
	engine := server.New(db.Connection, sqlDB, version)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("starting server", slog.String("addr", srv.Addr), slog.String("env", cfg.Env))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.LogAndPanic(err, "server failed")
		}
	}()

	sig := <-quit
	slog.Info("shutting down", slog.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", slog.String("error", err.Error()))
	}

	slog.Info("server stopped")
}
