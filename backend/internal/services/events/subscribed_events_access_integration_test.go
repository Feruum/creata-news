//go:build integration

package events

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	authzed "github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/perms"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestUserSubscribedEventsRequireOwnerOrCurrentPlatformStaff(t *testing.T) {
	if os.Getenv("RUN_SPICEDB_INTEGRATION_TESTS") != "1" {
		t.Skip("explicit disposable SpiceDB opt-in required")
	}
	endpoint, key := os.Getenv("SPICEDB_TEST_ENDPOINT"), os.Getenv("SPICEDB_TEST_PRESHARED_KEY")
	if endpoint == "" || key == "" {
		t.Fatal("disposable SpiceDB endpoint and key required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	schemaPath := os.Getenv("SPICEDB_TEST_SCHEMA_PATH")
	if schemaPath == "" {
		schemaPath = filepath.Join("..", "..", "..", "..", "..", "infra", "spicedb", "schema.zed")
	}
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := authzed.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpcutil.WithInsecureBearerToken(key))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.WriteSchema(ctx, &pb.WriteSchemaRequest{Schema: string(schema)}); err != nil {
		t.Fatal(err)
	}
	graph, err := perms.NewClient(endpoint, key, true, false, fmt.Sprintf("subscription-access-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	db := newTestDB(t)
	owner := seedAuthUser(t, db, "subscription-owner")
	other := seedAuthUser(t, db, "subscription-other")
	staff := seedAuthUser(t, db, "subscription-staff")
	admin := seedAuthUser(t, db, "subscription-admin")
	for userID, role := range map[int32]string{owner: "none", other: "none", staff: "staff", admin: "administrator"} {
		if _, err := db.ExecContext(ctx, "INSERT INTO platform_user_roles(user_id,role) VALUES(?,?)", userID, role); err != nil {
			t.Fatal(err)
		}
		if role != "none" {
			if err := graph.SetPlatformRole(ctx, fmt.Sprint(userID), role); err != nil {
				t.Fatal(err)
			}
		}
	}
	org := seedOrganization(t, db, "Private subscriptions club")
	event := seedDraftEvent(t, db, owner, org.ID, "Owner participation", "Hall")
	registration := &models.EventRegistration{EventID: event.ID, UserID: owner, Status: models.RegistrationStatus("registered"), RegisteredAt: time.Now()}
	if _, err := db.NewInsert().Model(registration).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, graph, authz.NewEnforcer(db, graph), nil, nil, "", "")
	for _, test := range []struct {
		name, identity string
		code           connect.Code
	}{
		{name: "anonymous denied", code: connect.CodeUnauthenticated},
		{name: "different ordinary user denied", identity: "subscription-other", code: connect.CodePermissionDenied},
		{name: "owner allowed", identity: "subscription-owner"},
		{name: "platform staff allowed", identity: "subscription-staff"},
		{name: "platform administrator allowed", identity: "subscription-admin"},
	} {
		t.Run(test.name, func(t *testing.T) {
			requestCtx := ctx
			if test.identity != "" {
				requestCtx = authedContext(test.identity)
			}
			response, err := svc.GetUserSubscribedEvents(requestCtx, connect.NewRequest(&eventsv1.GetUserSubscribedEventsRequest{UserId: owner}))
			if test.code != 0 {
				if connect.CodeOf(err) != test.code || response != nil {
					t.Fatalf("participation exposed: response=%v err=%v want=%v", response, err, test.code)
				}
				return
			}
			if err != nil || len(response.Msg.Events) != 1 || response.Msg.Events[0].Id != event.ID {
				t.Fatalf("authorized participation unavailable: response=%v err=%v", response, err)
			}
		})
	}
	t.Run("revoked staff denied despite stale graph", func(t *testing.T) {
		if _, err := db.ExecContext(ctx, "UPDATE platform_user_roles SET role='none' WHERE user_id=?", staff); err != nil {
			t.Fatal(err)
		}
		response, err := svc.GetUserSubscribedEvents(authedContext("subscription-staff"), connect.NewRequest(&eventsv1.GetUserSubscribedEventsRequest{UserId: owner}))
		if connect.CodeOf(err) != connect.CodePermissionDenied || response != nil {
			t.Fatalf("revoked staff exposed participation: response=%v err=%v", response, err)
		}
	})
}
