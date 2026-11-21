//go:build integration

package statistics

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

type staleStatisticsGraph struct {
	pb.UnimplementedPermissionsServiceServer
}

func (*staleStatisticsGraph) CheckPermission(context.Context, *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	return &pb.CheckPermissionResponse{Permissionship: pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION}, nil
}

func TestStatisticsRequiresCurrentPlatformOrScopedAnalyticsGrant(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	insert := func(query string, args ...any) int32 {
		t.Helper()
		var id int32
		if err := db.NewRaw(query, args...).Scan(ctx, &id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	actor := func(name, role string) (int32, context.Context) {
		id := insert("INSERT INTO users(kratos_id,username,email,password) VALUES(?,?,?,'test') RETURNING id", name, name, name+"@test.local")
		exec("INSERT INTO platform_user_roles(user_id,role) VALUES(?,?)", id, role)
		return id, context.WithValue(ctx, auth.UserIDKey, name)
	}
	regular, regularCtx := actor("audit-stats-regular", "none")
	_, staffCtx := actor("audit-stats-staff", "staff")
	creator, creatorCtx := actor("audit-stats-creator", "none")
	analyst, analystCtx := actor("audit-stats-analyst", "none")
	typ := insert("INSERT INTO organization_types(title) VALUES('Audit statistics type') RETURNING id")
	org := insert("INSERT INTO organizations(title,organization_type_id) VALUES('Audit statistics org',?) RETURNING id", typ)
	otherOrg := insert("INSERT INTO organizations(title,organization_type_id) VALUES('Other statistics org',?) RETURNING id", typ)
	role := insert("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'statistics-analyst','Analyst',true) RETURNING id", typ)
	exec("INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'view_analytics')", role)
	exec("INSERT INTO organization_members(organization_id,user_id,role_template_id,assigned_by) VALUES(?,?,?,?)", org, analyst, role, creator)
	event := insert("INSERT INTO events(title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token) VALUES('Private statistics event','{}',?,?,NOW(),NOW()+interval '1 hour','{}','draft','stats-token','stats-token') RETURNING id", creator, org)
	reg := insert("INSERT INTO event_registrations(event_id,user_id,status) VALUES(?,?,'registered') RETURNING id", event, regular)
	exec("INSERT INTO event_attendance(registration_id,status,checked_in_at,method) VALUES(?,'attended',NOW(),'auto')", reg)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, &staleStatisticsGraph{})
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(db, authz.NewEnforcer(db, client))
	for name, call := range statisticsCalls(service) {
		if name == "event" || name == "org trends" {
			continue
		}
		t.Run("ordinary denied global/"+name, func(t *testing.T) {
			if err := call(regularCtx); connect.CodeOf(err) != connect.CodePermissionDenied {
				t.Fatalf("ordinary user read global analytics: %v", err)
			}
		})
	}
	if _, err := service.GetOverallStatistics(staffCtx, connect.NewRequest(&eventsv1.GetOverallStatisticsRequest{})); err != nil {
		t.Fatalf("current staff denied global statistics: %v", err)
	}
	eventRequest := connect.NewRequest(&eventsv1.GetEventStatisticsRequest{EventId: event})
	for name, actorCtx := range map[string]context.Context{"staff": staffCtx, "creator": creatorCtx, "current analyst": analystCtx} {
		t.Run(name+" event scope", func(t *testing.T) {
			response, err := service.GetEventStatistics(actorCtx, eventRequest)
			if err != nil || response.Msg.TotalRegistrations != 1 || response.Msg.TotalAttendees != 1 {
				t.Fatalf("valid scoped analytics denied/incomplete: response=%v error=%v", response, err)
			}
		})
	}
	if _, err := service.GetEventStatistics(regularCtx, eventRequest); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("ordinary attendee read private event statistics: %v", err)
	}
	if _, err := service.GetOrganizationTrends(analystCtx, connect.NewRequest(&eventsv1.GetOrganizationTrendsRequest{OrganizationIds: []int32{org}, Days: 90})); err != nil {
		t.Fatalf("organization analyst denied own trends: %v", err)
	}
	if _, err := service.GetOrganizationTrends(analystCtx, connect.NewRequest(&eventsv1.GetOrganizationTrendsRequest{OrganizationIds: []int32{org, otherOrg}, Days: 90})); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("mixed scope exposed unauthorized organization: %v", err)
	}
	for _, state := range []string{"expired", "deleted"} {
		if state == "expired" {
			exec("UPDATE organization_members SET valid_until=? WHERE organization_id=? AND user_id=?", time.Now().Add(-time.Hour), org, analyst)
		} else {
			exec("DELETE FROM organization_members WHERE organization_id=? AND user_id=?", org, analyst)
		}
		if _, err := service.GetEventStatistics(analystCtx, eventRequest); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("%s SQL member retained stale graph analytics: %v", state, err)
		}
	}
	// Authorized requests must return storage failures instead of fabricated zeros.
	exec("ALTER TABLE event_attendance RENAME TO unavailable_event_attendance")
	if _, err := service.GetEventStatistics(creatorCtx, eventRequest); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("attendance database failure reported success: %v", err)
	}
	t.Log(fmt.Sprintf("guarded private event %d, organization %d and all global RPCs", event, org))
}
