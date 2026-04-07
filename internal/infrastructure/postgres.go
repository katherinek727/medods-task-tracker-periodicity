package infrastructure

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds Postgres connection parameters.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewPostgresPool creates and validates a pgxpool connection pool.
func NewPostgresPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres: failed to ping: %w", err)
	}

	return pool, nil
}
