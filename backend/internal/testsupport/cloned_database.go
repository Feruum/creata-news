//go:build integration

package testsupport

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// ClonedDB creates a disposable copy of the explicitly configured test database.
func ClonedDB(t *testing.T) *bun.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL must name a disposable migrated test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	source := strings.TrimPrefix(parsed.Path, "/")
	if source == "" {
		t.Fatal("DATABASE_URL requires a database name")
	}
	parsed.Path = "/postgres"
	admin := bun.NewDB(sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(parsed.String()))), pgdialect.New())
	name := fmt.Sprintf("audit_clone_%d", time.Now().UnixNano())
	quote := func(value string) string { return `"` + strings.ReplaceAll(value, `"`, `""`) + `"` }
	if _, err := admin.ExecContext(context.Background(), "CREATE DATABASE "+quote(name)+" TEMPLATE "+quote(source)); err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	parsed.Path = "/" + name
	db := bun.NewDB(sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(parsed.String()))), pgdialect.New())
	t.Cleanup(func() {
		_ = db.Close()
		_, _ = admin.ExecContext(context.Background(), "DROP DATABASE "+quote(name)+" WITH (FORCE)")
		_ = admin.Close()
	})
	return db
}
