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
	kafkapkg "github.com/getlago/lago/api-go/internal/kafka"
	"github.com/getlago/lago/api-go/internal/observability"
	"github.com/getlago/lago/api-go/internal/server"
	"github.com/getlago/lago/api-go/pkg/env"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid config", slog.String("error", err.Error()))
		observability.CaptureError(err)
		panic(err.Error())
	}

	logLevel := slog.LevelInfo
	if cfg.Env == "development" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(
		observability.NewLevelHandler(
			logLevel,
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}),
		),
	).With("service", "lago-api")
	slog.SetDefault(logger)

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.Observability.SentryDSN,
		Environment:      cfg.Env,
		AttachStacktrace: true,
	}); err != nil {
		slog.Warn("sentry init failed", slog.String("error", err.Error()))
	}
	defer sentry.Flush(2 * time.Second)

	db, err := database.NewConnection(database.Config{
		URL:      cfg.Database.URL,
		MaxConns: cfg.Database.MaxConns,
	})
	if err != nil {
		slog.Error("failed to connect to database", slog.String("error", err.Error()))
		observability.CaptureError(err)
		panic(err.Error())
	}
	defer db.Close()

	sqlDB, err := db.Connection.DB()
	if err != nil {
		slog.Error("failed to get sql.DB from gorm", slog.String("error", err.Error()))
		observability.CaptureError(err)
		panic(err.Error())
	}

	redisClient, redisErr := cfgredis.NewRedisDB(context.Background(), cfgredis.Config{
		Address:  cfg.Redis.URL,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		UseTLS:   cfg.Redis.UseTLS,
	})
	if redisErr != nil {
		slog.Warn("failed to connect to redis", slog.String("error", redisErr.Error()))
	} else {
		defer redisClient.Close()
	}

	version := env.GetEnvOrDefault("APP_VERSION", "dev")
	eventPublisher := buildEventPublisher(cfg)
	engine := server.New(db.Connection, sqlDB, version, cfg.Auth.JWTSecret, eventPublisher)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTP.Port),
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
			slog.Error("server failed", slog.String("error", err.Error()))
			observability.CaptureError(err)
			panic(err.Error())
		}
	}()

	sig := <-quit
	slog.Info("shutting down", slog.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", slog.String("error", err.Error()))
	}

	if publisher, ok := eventPublisher.(*kafkapkg.KafkaPublisher); ok {
		publisher.Close()
		slog.Info("kafka publisher closed")
	}

	slog.Info("server stopped")
}

func buildEventPublisher(cfg *config.Config) kafkapkg.EventPublisher {
	if cfg.Kafka.BootstrapServers == "" {
		slog.Info("kafka not configured, events will not be published to kafka")
		return &kafkapkg.NoopPublisher{}
	}

	publisher, err := kafkapkg.NewKafkaPublisher(kafkapkg.Config{
		BootstrapServers: cfg.Kafka.BootstrapServers,
		Topic:            cfg.Kafka.RawEventsTopic,
		TLS:              cfg.Kafka.TLS,
		ScramAlgorithm:   cfg.Kafka.ScramAlgorithm,
		Username:         cfg.Kafka.Username,
		Password:         cfg.Kafka.Password,
	})
	if err != nil {
		slog.Warn("failed to initialize kafka producer, falling back to noop", slog.String("error", err.Error()))
		return &kafkapkg.NoopPublisher{}
	}

	slog.Info("kafka producer initialized",
		slog.String("bootstrap_servers", cfg.Kafka.BootstrapServers),
		slog.String("topic", cfg.Kafka.RawEventsTopic),
	)
	return publisher
}
