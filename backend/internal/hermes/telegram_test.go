package hermes

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// An in-process SQL driver exercises Bun's actual user selection and row mapping without a live database.
type telegramDB struct {
	query  string
	values []driver.Value
	err    error
}

func (d *telegramDB) Connect(context.Context) (driver.Conn, error) { return d, nil }
func (d *telegramDB) Driver() driver.Driver                        { return d }
func (d *telegramDB) Open(string) (driver.Conn, error)             { return d, nil }
func (d *telegramDB) Close() error                                 { return nil }
func (d *telegramDB) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepared statement")
}
func (d *telegramDB) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }
func (d *telegramDB) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	d.query = query
	if d.err != nil {
		return nil, d.err
	}
	return &telegramRows{values: d.values}, nil
}

type telegramRows struct {
	values []driver.Value
	read   bool
}

func (r *telegramRows) Columns() []string { return []string{"telegram_id", "telegram_linked_at"} }
func (r *telegramRows) Close() error      { return nil }
func (r *telegramRows) Next(values []driver.Value) error {
	if r.read || r.values == nil {
		return io.EOF
	}
	r.read = true
	copy(values, r.values)
	return nil
}

func TestTelegramStoreUsesKratosIDAndOnlyVerifiedNumericLink(t *testing.T) {
	linkedAt := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name         string
		values       []driver.Value
		err, wantErr error
	}{
		{"verified", []driver.Value{int64(987654321), linkedAt}, nil, nil},
		{"unlinked", []driver.Value{nil, nil}, nil, ErrDenied},
		{"not verified", []driver.Value{int64(987654321), nil}, nil, ErrDenied},
		{"group id", []driver.Value{int64(-1001234), linkedAt}, nil, ErrDenied},
		{"future link", []driver.Value{int64(987654321), time.Now().Add(time.Hour)}, nil, ErrDenied},
		{"missing user", nil, nil, ErrDenied},
		{"DB outage", nil, errors.New("private-database-diagnostic"), ErrUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fake := &telegramDB{values: tt.values, err: tt.err}
			db := bun.NewDB(sql.OpenDB(fake), pgdialect.New())
			t.Cleanup(func() { _ = db.Close() })
			link, err := NewTelegramStore(db).VerifiedLink(context.Background(), "owner-kratos-id")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && (link.ID != "987654321" || !link.LinkedAt.Equal(linkedAt)) {
				t.Fatalf("link = %#v", link)
			}
			if !strings.Contains(fake.query, "kratos_id") || !strings.Contains(fake.query, "owner-kratos-id") || strings.Contains(fake.query, "email") {
				t.Fatalf("lookup used wrong identity boundary: %s", fake.query)
			}
		})
	}
}
