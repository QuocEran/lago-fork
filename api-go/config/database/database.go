package database

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DB struct {
	Connection *gorm.DB
	pool       *pgxpool.Pool
}

type Config struct {
	URL      string
	MaxConns int32
}

func NewConnection(cfg Config) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, err
	}
	poolConfig.MaxConns = cfg.MaxConns

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	dialector := postgres.New(postgres.Config{
		Conn: stdlib.OpenDBFromPool(pool),
	})

	conn, err := openConnection(dialector)
	if err != nil {
		pool.Close()
		return nil, err
	}
	conn.pool = pool
	return conn, nil
}

func openConnection(dialector gorm.Dialector) (*DB, error) {
	logger := slog.Default().With("component", "db")
	gormLogger := slogGorm.New(
		slogGorm.WithHandler(logger.Handler()),
	)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	return &DB{Connection: db}, nil
}

func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}
