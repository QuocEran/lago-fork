package redis

import (
	"context"
	"crypto/tls"
	"regexp"
	"time"

	redisotel "github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Address   string
	Password  string
	DB        int
	UseTracer bool
	UseTLS    bool
}

type RedisDB struct {
	Client *redis.Client
}

func (c *Config) addressAndPort() string {
	re := regexp.MustCompile(`^rediss?://`)
	return re.ReplaceAllString(c.Address, "")
}

func NewRedisDB(ctx context.Context, cfg Config) (*RedisDB, error) {
	opts := &redis.Options{
		Addr:         cfg.addressAndPort(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		PoolTimeout:  4 * time.Second,
	}

	if cfg.UseTLS {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	if cfg.UseTracer {
		if err := redisotel.InstrumentTracing(client); err != nil {
			return nil, err
		}
	}

	return &RedisDB{Client: client}, nil
}

// Close releases the Redis connection pool.
func (r *RedisDB) Close() error {
	return r.Client.Close()
}
