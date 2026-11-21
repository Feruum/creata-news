package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string, enableTracing bool) (*pgxpool.Pool, error) {
	return NewPoolWithMaxConns(ctx, databaseURL, enableTracing, 0)
}

// NewPoolWithMaxConns creates a pool with a bounded connection count when
// maxConns is positive. Leaving it at zero keeps the pgx default, which is
// useful for focused tests that should not inherit production sizing.
func NewPoolWithMaxConns(ctx context.Context, databaseURL string, enableTracing bool, maxConns int32) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}
	if maxConns > 0 {
		config.MaxConns = maxConns
	}

	if enableTracing {
		// Bun interpolates its arguments before executing through pgx. Keep SQL
		// templates in the Bun span; the lower layer must never export those
		// expanded values in either query attributes or full-query span names.
		config.ConnConfig.Tracer = otelpgx.NewTracer(
			otelpgx.WithDisableSQLStatementInAttributes(),
			otelpgx.WithTrimSQLInSpanName(),
		)
		slog.Info("Database tracing enabled")
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Connected to database", "max_conns", config.MaxConns)
	return pool, nil
}
