//go:build integration

package rooms

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
	"google.golang.org/grpc"
)

func bookingDedupDB(t *testing.T) *bun.DB {
	t.Helper()
	db := testsupport.IsolatedDB(t)
	_, err := db.ExecContext(context.Background(), `
		CREATE TABLE users (LIKE public.users INCLUDING ALL);
		CREATE TABLE organizations (LIKE public.organizations INCLUDING ALL);
		CREATE TABLE organization_types (LIKE public.organization_types INCLUDING ALL);
		CREATE TABLE platform_user_roles (LIKE public.platform_user_roles INCLUDING ALL);
		CREATE TABLE events (LIKE public.events INCLUDING ALL);
		CREATE TABLE rooms (LIKE public.rooms INCLUDING ALL);
		CREATE TABLE booking_requests (LIKE public.booking_requests INCLUDING ALL);
		CREATE TABLE room_bookings (LIKE public.room_bookings INCLUDING ALL);
		CREATE SEQUENCE booking_requests_id_seq OWNED BY booking_requests.id;
		ALTER TABLE booking_requests ALTER COLUMN id SET DEFAULT nextval('booking_requests_id_seq');
		CREATE SEQUENCE room_bookings_id_seq OWNED BY room_bookings.id;
		ALTER TABLE room_bookings ALTER COLUMN id SET DEFAULT nextval('room_bookings_id_seq');
		ALTER TABLE booking_requests ADD FOREIGN KEY(event_id) REFERENCES events(id);
		INSERT INTO users(id,kratos_id,username,email,password) VALUES(1,'booking-dedup-qa','booking-dedup-qa','booking-qa@example.test','synthetic-not-for-login');
		INSERT INTO organization_types(id,title) VALUES(1,'Student Club');
		INSERT INTO organizations(id,title,organization_type_id) VALUES(1,'Booking QA',1);
		INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token)
		VALUES(1,'Booking QA','{}',1,1,'2026-10-20T10:00:00Z','2026-10-20T11:00:00Z','{}','draft','booking-qa-secret','booking-qa-token');
		INSERT INTO rooms(id,name,room_number,block,floor) VALUES('C1.3.334L','Booking QA room','334L','C1',3);`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func bookingDedupRequest(room string) *connect.Request[roomsv1.CreateBookingRequestRequest] {
	eventID, organizationID := int32(1), int32(1)
	return connect.NewRequest(&roomsv1.CreateBookingRequestRequest{
		RoomId: room, EventId: &eventID, OrganizationId: &organizationID, Title: "Booking QA",
		StartTime: "2026-10-20T10:00:00Z", EndTime: "2026-10-20T11:00:00Z",
	})
}

func TestConcurrentLinkedBookingRequestsCreateOneActiveRequest(t *testing.T) {
	for _, expected := range []models.BookingRequestStatus{models.BookingRequestStatusPending, models.BookingRequestStatusApproved} {
		t.Run(string(expected), func(t *testing.T) {
			db := bookingDedupDB(t)
			ctx, cancel := context.WithTimeout(authedContext("booking-dedup-qa"), 10*time.Second)
			defer cancel()
			if expected == models.BookingRequestStatusPending {
				// Existing confirmed occupancy preserves the staff auto-approval policy.
				if _, err := db.ExecContext(ctx, `INSERT INTO room_bookings(id,room_id,created_by,title,start_time,end_time)
					VALUES(100,'C1.3.334L',1,'Other occupancy','2026-10-20T10:00:00Z','2026-10-20T11:00:00Z')`); err != nil {
					t.Fatal(err)
				}
			}
			svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil)
			start := make(chan struct{})
			results := make(chan error, 2)
			for _, room := range []string{" c1.3.334l ", "C1.3.334L"} {
				go func(room string) {
					<-start
					_, err := svc.CreateBookingRequest(ctx, bookingDedupRequest(room))
					results <- err
				}(room)
			}
			close(start)
			succeeded, duplicates := 0, 0
			for range 2 {
				select {
				case err := <-results:
					switch {
					case err == nil:
						succeeded++
					case connect.CodeOf(err) == connect.CodeAlreadyExists:
						duplicates++
						if !strings.Contains(err.Error(), "already exists") {
							t.Errorf("duplicate error must explain the existing request: %v", err)
						}
					default:
						t.Errorf("unexpected create error: %v", err)
					}
				case <-ctx.Done():
					t.Fatal("concurrent booking requests did not complete")
				}
			}
			count, err := db.NewSelect().Model((*models.BookingRequest)(nil)).
				Where("event_id = 1 AND room_id = 'C1.3.334L'").
				Where("start_time = ? AND end_time = ?", "2026-10-20T10:00:00Z", "2026-10-20T11:00:00Z").
				Where("status IN (?, ?)", models.BookingRequestStatusPending, models.BookingRequestStatusApproved).Count(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if count != 1 || succeeded != 1 || duplicates != 1 {
				t.Fatalf("active requests=%d, successful creates=%d, already_exists=%d; want 1 each", count, succeeded, duplicates)
			}
			var request models.BookingRequest
			if err := db.NewSelect().Model(&request).Where("event_id = 1").Scan(ctx); err != nil {
				t.Fatal(err)
			}
			if request.Status != expected {
				t.Fatalf("status=%s, want %s without changing approval policy", request.Status, expected)
			}
		})
	}
}

func TestRejectedLinkedBookingRequestAllowsNewRequest(t *testing.T) {
	db := bookingDedupDB(t)
	ctx := authedContext("booking-dedup-qa")
	if _, err := db.ExecContext(ctx, `INSERT INTO booking_requests(id,room_id,requester_id,organization_id,event_id,title,start_time,end_time,status)
		VALUES(100,'C1.3.334L',1,1,1,'Rejected QA','2026-10-20T10:00:00Z','2026-10-20T11:00:00Z','rejected')`); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil)
	response, err := svc.CreateBookingRequest(ctx, bookingDedupRequest("C1.3.334L"))
	if err != nil {
		t.Fatalf("rejected request must permit a new request: %v", err)
	}
	if response.Msg.BookingRequest.Id == 100 {
		t.Fatal("new request reused the rejected request")
	}
	count, err := db.NewSelect().Model((*models.BookingRequest)(nil)).Where("event_id = 1").Count(ctx)
	if err != nil || count != 2 {
		t.Fatalf("request history count=%d, error=%v; want rejected plus new request", count, err)
	}
}

func TestBookingDedupPreservesDifferentIntervalsAndUnlinkedRequests(t *testing.T) {
	for _, scenario := range []string{"different_interval", "unlinked"} {
		t.Run(scenario, func(t *testing.T) {
			db := bookingDedupDB(t)
			ctx := authedContext("booking-dedup-qa")
			if _, err := db.ExecContext(ctx, `INSERT INTO booking_requests(id,room_id,requester_id,organization_id,event_id,title,start_time,end_time,status)
				VALUES(100,'C1.3.334L',1,1,1,'Existing QA','2026-10-20T10:00:00Z','2026-10-20T11:00:00Z','pending')`); err != nil {
				t.Fatal(err)
			}
			svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil)
			request := bookingDedupRequest("C1.3.334L")
			if scenario == "different_interval" {
				request.Msg.StartTime, request.Msg.EndTime = "2026-10-20T11:00:00Z", "2026-10-20T12:00:00Z"
			} else {
				request.Msg.EventId = nil
			}
			if _, err := svc.CreateBookingRequest(ctx, request); err != nil {
				t.Fatalf("valid %s request rejected: %v", scenario, err)
			}
			if scenario == "unlinked" {
				if _, err := svc.CreateBookingRequest(ctx, request); err != nil {
					t.Fatalf("unlinked repeats must preserve existing behavior: %v", err)
				}
			}
		})
	}
}

type delayedBookingStaffPermission struct {
	pb.UnimplementedPermissionsServiceServer
	mu      sync.Mutex
	calls   int
	started chan struct{}
	release chan struct{}
}

func (p *delayedBookingStaffPermission) CheckPermission(ctx context.Context, req *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	if req.Resource.ObjectType == "platform" {
		p.mu.Lock()
		p.calls++
		lastStaffCheck := p.calls == 2
		p.mu.Unlock()
		if lastStaffCheck {
			close(p.started)
			select {
			case <-p.release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	return &pb.CheckPermissionResponse{Permissionship: pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION}, nil
}

func TestLinkedBookingStaffAuthorizationDoesNotHoldEventRowLock(t *testing.T) {
	db := bookingDedupDB(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	permission := &delayedBookingStaffPermission{started: make(chan struct{}), release: make(chan struct{})}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, permission)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test-only", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(authedContext("booking-dedup-qa"), 10*time.Second)
	defer cancel()
	svc := NewService(db, nil, authz.NewEnforcer(db, client), nil)
	done := make(chan error, 1)
	go func() {
		_, err := svc.CreateBookingRequest(ctx, bookingDedupRequest("C1.3.334L"))
		done <- err
	}()
	defer func() {
		close(permission.release)
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("booking failed after permission release: %v", err)
			}
		case <-ctx.Done():
			t.Error("booking did not complete after permission release")
		}
	}()
	select {
	case <-permission.started:
	case <-ctx.Done():
		t.Fatal("delayed staff authorization did not start")
	}
	lockCtx, lockCancel := context.WithTimeout(ctx, 2*time.Second)
	defer lockCancel()
	if err := db.RunInTx(lockCtx, nil, func(ctx context.Context, tx bun.Tx) error {
		var id int32
		return tx.NewRaw("SELECT id FROM events WHERE id=1 FOR UPDATE").Scan(ctx, &id)
	}); err != nil {
		t.Fatal(fmt.Errorf("event mutation lock blocked by network authorization: %w", err))
	}
}
