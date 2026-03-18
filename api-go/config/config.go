package config

import (
	"os"
	"strconv"
)

type Config struct {
	Env              string
	DatabaseURL      string
	DatabaseMaxConns int32
	RedisURL         string
	RedisPassword    string
	RedisDB          int
	RedisTLS         bool
	JWTSecret        string
	SentryDSN        string
	OtelEndpoint     string
	Port             string
}

func Load() *Config {
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	return &Config{
		Env:              getEnv("ENV", "development"),
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		DatabaseMaxConns: int32(getEnvAsInt("DATABASE_POOL", 10)),
		RedisURL:         redisURL,
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		RedisDB:          getEnvAsInt("REDIS_DB", 0),
		RedisTLS:         len(redisURL) > 8 && redisURL[:8] == "rediss://",
		JWTSecret:        getEnv("SECRET_KEY_BASE", ""),
		SentryDSN:        getEnv("SENTRY_DSN", ""),
		OtelEndpoint:     getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		Port:             getEnv("PORT", "3000"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return i
}
