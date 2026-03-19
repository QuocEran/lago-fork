package config

import (
	"fmt"

	"github.com/getlago/lago/api-go/pkg/env"
)

// HTTPConfig holds HTTP server settings.
type HTTPConfig struct {
	Port string
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	URL      string
	MaxConns int32
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	URL      string
	Password string
	DB       int
	UseTLS   bool
}

// KafkaConfig holds Kafka producer settings.
type KafkaConfig struct {
	BootstrapServers string
	RawEventsTopic   string
	TLS              bool
	ScramAlgorithm   string
	Username         string
	Password         string
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTSecret string
}

// ObservabilityConfig holds Sentry and OpenTelemetry settings.
type ObservabilityConfig struct {
	SentryDSN    string
	OtelEndpoint string
}

// Config holds all application configuration.
type Config struct {
	Env           string
	HTTP          HTTPConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	Kafka         KafkaConfig
	Auth          AuthConfig
	Observability ObservabilityConfig
}

// Validate checks required configuration and returns an error if invalid.
// Call after Load() to fail fast at startup.
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required and must not be empty")
	}
	return nil
}

// Load reads configuration from the environment.
func Load() *Config {
	redisURL := env.GetEnvOrDefault("REDIS_URL", "redis://localhost:6379")
	dbPool, _ := env.GetEnvAsInt("DATABASE_POOL", 10)
	redisDB, _ := env.GetEnvAsInt("REDIS_DB", 0)

	return &Config{
		Env: env.GetEnvOrDefault("ENV", "development"),
		HTTP: HTTPConfig{
			Port: env.GetEnvOrDefault("PORT", "3000"),
		},
		Database: DatabaseConfig{
			URL:      env.GetEnvOrDefault("DATABASE_URL", ""),
			MaxConns: int32(dbPool),
		},
		Redis: RedisConfig{
			URL:      redisURL,
			Password: env.GetEnvOrDefault("REDIS_PASSWORD", ""),
			DB:       redisDB,
			UseTLS:   len(redisURL) > 8 && redisURL[:8] == "rediss://",
		},
		Kafka: KafkaConfig{
			BootstrapServers: env.GetEnvOrDefault("LAGO_KAFKA_BOOTSTRAP_SERVERS", ""),
			RawEventsTopic:   env.GetEnvOrDefault("LAGO_KAFKA_RAW_EVENTS_TOPIC", "events-raw"),
			TLS:              env.GetEnvOrDefault("LAGO_KAFKA_TLS", "") == "true",
			ScramAlgorithm:   env.GetEnvOrDefault("LAGO_KAFKA_SCRAM_ALGORITHM", ""),
			Username:         env.GetEnvOrDefault("LAGO_KAFKA_USERNAME", ""),
			Password:         env.GetEnvOrDefault("LAGO_KAFKA_PASSWORD", ""),
		},
		Auth: AuthConfig{
			JWTSecret: env.GetEnvOrDefault("SECRET_KEY_BASE", ""),
		},
		Observability: ObservabilityConfig{
			SentryDSN:    env.GetEnvOrDefault("SENTRY_DSN", ""),
			OtelEndpoint: env.GetEnvOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		},
	}
}
