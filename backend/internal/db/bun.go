package db

import (
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bunotel"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
)

// NewBunDB creates a new Bun DB instance from an existing pgxpool.Pool.
// It bridges the pgx pool into database/sql via stdlib, then wraps it with Bun.
func NewBunDB(pool *pgxpool.Pool, enableTracing bool) *bun.DB {
	sqldb := stdlib.OpenDBFromPool(pool)
	db := bun.NewDB(sqldb, pgdialect.New())

	if enableTracing {
		db.AddQueryHook(bunotel.NewQueryHook(
			bunotel.WithDBName("eventdb"),
			bunotel.WithFormattedQueries(false),
		))

		otelsql.ReportDBStatsMetrics(sqldb, otelsql.WithDBSystem("postgresql"))
	}

	return db
}

// CloseBunDB closes only the Bun wrapper and its underlying sql.DB.
// The pgxpool.Pool should be closed separately (it owns the actual connections).
func CloseBunDB(db *bun.DB) error {
	if db == nil {
		return nil
	}
	sqlDB := db.DB
	// Close the Bun wrapper
	if err := db.Close(); err != nil {
		return err
	}
	// Close the sql.DB (releases stdlib adapter)
	_ = sqlDB
	return nil
}

// DB returns the underlying *sql.DB from a Bun instance.
func DB(bunDB *bun.DB) *sql.DB {
	return bunDB.DB
}
