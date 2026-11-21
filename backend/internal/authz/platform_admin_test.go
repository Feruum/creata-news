package authz

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type adminLookupConnector struct {
	userErr error
	roleErr error
	role    string
}

func (c adminLookupConnector) Connect(context.Context) (driver.Conn, error) {
	return adminLookupConn{c}, nil
}
func (c adminLookupConnector) Driver() driver.Driver { return adminLookupDriver{c} }

type adminLookupDriver struct{ connector adminLookupConnector }

func (d adminLookupDriver) Open(string) (driver.Conn, error) {
	return d.connector.Connect(context.Background())
}

type adminLookupConn struct{ connector adminLookupConnector }

func (c adminLookupConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c adminLookupConn) Close() error              { return nil }
func (c adminLookupConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }
func (c adminLookupConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, `FROM "users"`) {
		if strings.Contains(query, `"avatar_url"`) {
			return nil, errors.New(`column "avatar_url" does not exist`)
		}
		if c.connector.userErr != nil {
			return nil, c.connector.userErr
		}
		return &adminLookupRows{column: "id", value: int64(42)}, nil
	}
	if strings.Contains(query, "platform_user_roles") {
		if c.connector.roleErr != nil {
			return nil, c.connector.roleErr
		}
		return &adminLookupRows{column: "role", value: c.connector.role}, nil
	}
	return nil, errors.New("unexpected query")
}

type adminLookupRows struct {
	column string
	value  driver.Value
	done   bool
}

func (r *adminLookupRows) Columns() []string { return []string{r.column} }
func (r *adminLookupRows) Close() error      { return nil }
func (r *adminLookupRows) Next(values []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	values[0] = r.value
	return nil
}

type adminPermissionServer struct {
	pb.UnimplementedPermissionsServiceServer
	allowed bool
	err     error
}

func (s adminPermissionServer) CheckPermission(context.Context, *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	permission := pb.CheckPermissionResponse_PERMISSIONSHIP_NO_PERMISSION
	if s.allowed {
		permission = pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION
	}
	return &pb.CheckPermissionResponse{Permissionship: permission}, nil
}

func TestRequirePlatformAdminPreservesDependencyErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		lookup     adminLookupConnector
		graph      adminPermissionServer
		noGraph    bool
		wantCode   connect.Code
		wantUserID int32
	}{
		{name: "user database unavailable", lookup: adminLookupConnector{userErr: errors.New("database unavailable")}, wantCode: connect.CodeUnavailable},
		{name: "role database unavailable", lookup: adminLookupConnector{roleErr: errors.New("role database unavailable")}, wantCode: connect.CodeUnavailable},
		{name: "permission service unavailable", lookup: adminLookupConnector{role: "administrator"}, graph: adminPermissionServer{err: status.Error(codes.Unavailable, "service offline")}, wantCode: connect.CodeUnavailable},
		{name: "permission service missing", noGraph: true, wantCode: connect.CodeUnavailable},
		{name: "current staff denied", lookup: adminLookupConnector{role: "staff"}, graph: adminPermissionServer{allowed: true}, wantCode: connect.CodePermissionDenied},
		{name: "graph denial", lookup: adminLookupConnector{role: "administrator"}, wantCode: connect.CodePermissionDenied},
		{name: "administrator", lookup: adminLookupConnector{role: "administrator"}, graph: adminPermissionServer{allowed: true}, wantUserID: 42},
	} {
		t.Run(test.name, func(t *testing.T) {
			database := bun.NewDB(sql.OpenDB(test.lookup), pgdialect.New())
			t.Cleanup(func() { _ = database.Close() })
			var graph *perms.Client
			if !test.noGraph {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				server := grpc.NewServer()
				pb.RegisterPermissionsServiceServer(server, test.graph)
				go func() { _ = server.Serve(listener) }()
				t.Cleanup(server.Stop)
				graph, err = perms.NewClient(listener.Addr().String(), "test-key", true, false, "default")
				if err != nil {
					t.Fatal(err)
				}
			}
			ctx := context.WithValue(context.Background(), auth.UserIDKey, "administrator-id")
			userID, err := NewEnforcer(database, graph).RequirePlatformAdmin(ctx)
			if test.wantUserID != 0 {
				if err != nil || userID != test.wantUserID {
					t.Fatalf("administrator = %d %v", userID, err)
				}
			} else if err == nil || connect.CodeOf(err) != test.wantCode {
				t.Fatalf("error = %v, want %s", err, test.wantCode)
			}
		})
	}
}
