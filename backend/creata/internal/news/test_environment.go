package news

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/url"
	"regexp"
	"strings"
)

// Verification owns one unique schema; neither existing test data nor encrypted
// settings from another run are reused. This helper is never exposed by HTTP.
func IsolatedTestSchema(ctx context.Context, environment, dsn, name string, drop bool) (string, error) {
	u, e := url.Parse(dsn)
	if environment != "test" || e != nil || !isLoopback(u.Hostname()) || !strings.Contains(strings.ToLower(u.Path), "test") || !regexp.MustCompile(`^verify_[a-f0-9]{16}$`).MatchString(name) {
		return "", fmt.Errorf("verification schema requires test environment, loopback test database and a generated verify_ identifier")
	}
	pool, e := pgxpool.New(ctx, dsn)
	if e != nil {
		return "", e
	}
	defer pool.Close()
	statement := "CREATE SCHEMA " + pgx.Identifier{name}.Sanitize()
	if drop {
		statement = "DROP SCHEMA IF EXISTS " + pgx.Identifier{name}.Sanitize() + " CASCADE"
	}
	if _, e = pool.Exec(ctx, statement); e != nil {
		return "", e
	}
	q := u.Query()
	q.Set("search_path", name)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
