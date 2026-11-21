//go:build integration

package testsupport

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	internaldb "github.com/studyverse/ems-backend/internal/db"
	"github.com/uptrace/bun"
)

// IsolatedDB requires an explicitly configured disposable PostgreSQL database.
// Each test gets its own schema and connections; no production tables are mutated.
func IsolatedDB(t *testing.T) *bun.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL must point to a disposable migrated integration database")
	}
	ctx := context.Background()
	pool, err := internaldb.NewPool(ctx, dsn, false)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("audit_test_%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	params := parsed.Query()
	params.Set("search_path", schema+",public")
	parsed.RawQuery = params.Encode()
	testPool, err := internaldb.NewPool(ctx, parsed.String(), false)
	if err != nil {
		t.Fatal(err)
	}
	db := internaldb.NewBunDB(testPool, false)
	t.Cleanup(func() {
		_ = db.Close()
		testPool.Close()
		_, _ = pool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		pool.Close()
	})
	return db
}
