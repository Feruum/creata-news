//go:build integration

package orgmembers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

// capturePublisher records the last published topic and message.
type capturePublisher struct {
	topic string
	msg   *message.Message
}

func (p *capturePublisher) Publish(topic string, messages ...*message.Message) error {
	p.topic = topic
	if len(messages) > 0 {
		p.msg = messages[0]
	}
	return nil
}

func (p *capturePublisher) Close() error { return nil }

func newTestDB(t *testing.T) *bun.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/eventdb?sslmode=disable"
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	sourceDBName := strings.TrimPrefix(parsed.Path, "/")
	if sourceDBName == "" {
		t.Fatal("DATABASE_URL must include a database name")
	}

	adminURL := *parsed
	adminURL.Path = "/postgres"

	adminSQLDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(adminURL.String())))
	adminDB := bun.NewDB(adminSQLDB, pgdialect.New())
	tempDBName := fmt.Sprintf("orgmembers_svc_test_%d", time.Now().UnixNano())

	ctx := context.Background()
	createQuery := fmt.Sprintf(
		"CREATE DATABASE %s TEMPLATE %s",
		quoteIdentifier(tempDBName), quoteIdentifier(sourceDBName),
	)
	if _, err := adminDB.ExecContext(ctx, createQuery); err != nil {
		_ = adminDB.Close()
		t.Skipf("cannot create throwaway database (is Postgres running?): %v", err)
	}

	tempDSN := strings.Replace(parsed.String(), "/"+sourceDBName, "/"+tempDBName, 1)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(tempDSN)))
	db := bun.NewDB(sqldb, pgdialect.New())

	if _, err := db.ExecContext(ctx, "SET session_replication_role = 'replica'"); err != nil {
		_ = db.Close()
		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
		t.Fatalf("disable FK triggers: %v", err)
	}

	// Truncate relevant tables so tests start clean.
	for _, table := range []string{"organization_members", "organizations", "role_templates", "organization_types", "users", "membership_history"} {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", table)); err != nil {
			_ = db.Close()
			_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
			_ = adminDB.Close()
			t.Fatalf("truncate %s: %v", table, err)
		}
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "SET session_replication_role = 'origin'")
		_ = db.Close()
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
	})
	return db
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func authedContext(kratosID string) context.Context {
	return context.WithValue(context.Background(), auth.UserIDKey, kratosID)
}

// seedAuthUser inserts a user and returns its generated ID.
func seedAuthUser(t *testing.T, db *bun.DB, kratosID string) int32 {
	t.Helper()
	var id int32
	err := db.NewRaw(
		"INSERT INTO users (kratos_id, username, email, password) VALUES (?, ?, ?, ?) RETURNING id",
		kratosID, "seed-admin-"+kratosID, kratosID+"@test.local", "test",
	).Scan(context.Background(), &id)
	if err != nil {
		t.Fatalf("seed auth user: %v", err)
	}
	return id
}

// seedOrgAndRole creates an organization type, organization, and role template.
// Returns (orgID, roleTemplateID, orgTitle, roleName).
func seedOrgAndRole(t *testing.T, db *bun.DB) (int32, int32, string, string) {
	t.Helper()
	ctx := context.Background()

	var orgTypeID int32
	err := db.NewRaw("INSERT INTO organization_types (title) VALUES ('Test Type') RETURNING id").Scan(ctx, &orgTypeID)
	if err != nil {
		t.Fatalf("seed org type: %v", err)
	}

	var orgID int32
	err = db.NewRaw("INSERT INTO organizations (title, organization_type_id) VALUES ('Test Org', ?) RETURNING id", orgTypeID).Scan(ctx, &orgID)
	if err != nil {
		t.Fatalf("seed org: %v", err)
	}

	var roleID int32
	err = db.NewRaw(
		"INSERT INTO role_templates (organization_type_id, key, display_name, sort_order, is_required, is_active) VALUES (?, 'member', 'Member', 1, false, true) RETURNING id",
		orgTypeID,
	).Scan(ctx, &roleID)
	if err != nil {
		t.Fatalf("seed role template: %v", err)
	}

	return orgID, roleID, "Test Org", "Member"
}

func seedOrganizationMember(t *testing.T, db *bun.DB, orgID, userID, roleID, assignedBy int32) int32 {
	t.Helper()
	var memberID int32
	err := db.NewRaw(
		"INSERT INTO organization_members (organization_id, user_id, role_template_id, assigned_by) VALUES (?, ?, ?, ?) RETURNING id",
		orgID, userID, roleID, assignedBy,
	).Scan(context.Background(), &memberID)
	if err != nil {
		t.Fatalf("seed organization member: %v", err)
	}
	return memberID
}

func TestAddOrganizationMemberPublishesMemberAdded(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-orgmember"
	actorUserID := seedAuthUser(t, db, kratosID)
	targetUserID := seedAuthUser(t, db, "kratos-target-user")
	orgID, roleID, _, _ := seedOrgAndRole(t, db)

	cap := &capturePublisher{}
	pub := events.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t)) // nil perms → all permission checks pass

	svc := NewService(db, nil, enforcer, pub)
	ctx := authedContext(kratosID)

	resp, err := svc.AddOrganizationMember(ctx, connect.NewRequest(&eventsv1.AddOrganizationMemberRequest{
		OrganizationId: orgID,
		UserId:         targetUserID,
		RoleTemplateId: roleID,
		AssignedBy:     &actorUserID,
	}))
	if err != nil {
		t.Fatalf("AddOrganizationMember() error = %v", err)
	}

	// The publish must have been called with the right topic.
	if cap.topic != events.TopicMemberAdded {
		t.Fatalf("published topic = %q, want %q", cap.topic, events.TopicMemberAdded)
	}
	if cap.msg == nil {
		t.Fatal("expected a published message, got nil")
	}

	var got events.MemberAddedPayload
	if err := json.Unmarshal(cap.msg.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if got.UserID != targetUserID {
		t.Fatalf("payload UserID = %d, want %d", got.UserID, targetUserID)
	}
	if got.OrganizationID != orgID {
		t.Fatalf("payload OrganizationID = %d, want %d", got.OrganizationID, orgID)
	}
	if got.RoleTemplateID != roleID {
		t.Fatalf("payload RoleTemplateID = %d, want %d", got.RoleTemplateID, roleID)
	}
	if got.ActorID == nil || *got.ActorID != actorUserID {
		t.Fatalf("payload ActorID = %v, want %d", got.ActorID, actorUserID)
	}

	// Response should still be valid.
	if resp.Msg == nil || resp.Msg.Member == nil {
		t.Fatal("expected member response, got nil")
	}
	if resp.Msg.Member.UserId != targetUserID {
		t.Fatalf("response member UserId = %d, want %d", resp.Msg.Member.UserId, targetUserID)
	}
}

func TestRemoveOrganizationMemberDoesNotPublishWhenDeleteFails(t *testing.T) {
	db := newTestDB(t)
	if _, err := db.ExecContext(context.Background(), "SET session_replication_role = 'origin'"); err != nil {
		t.Fatalf("enable triggers: %v", err)
	}

	kratosID := "kratos-test-remove-orgmember"
	actorUserID := seedAuthUser(t, db, kratosID)
	targetUserID := seedAuthUser(t, db, "kratos-target-remove-user")
	orgID, roleID, _, _ := seedOrgAndRole(t, db)
	memberID := seedOrganizationMember(t, db, orgID, targetUserID, roleID, actorUserID)

	if _, err := db.ExecContext(context.Background(), `
		CREATE OR REPLACE FUNCTION fail_org_member_delete() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'forced delete failure';
		END;
		$$ LANGUAGE plpgsql;
	`); err != nil {
		t.Fatalf("create fail trigger function: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `
		CREATE TRIGGER organization_members_delete_fail
		BEFORE DELETE ON organization_members
		FOR EACH ROW EXECUTE FUNCTION fail_org_member_delete();
	`); err != nil {
		t.Fatalf("create fail trigger: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DROP TRIGGER IF EXISTS organization_members_delete_fail ON organization_members")
		_, _ = db.ExecContext(context.Background(), "DROP FUNCTION IF EXISTS fail_org_member_delete()")
	})

	cap := &capturePublisher{}
	pub := events.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, pub)

	_, err := svc.RemoveOrganizationMember(authedContext(kratosID), connect.NewRequest(&eventsv1.RemoveOrganizationMemberRequest{
		Id: memberID,
	}))
	if err == nil {
		t.Fatal("expected delete failure")
	}
	if cap.msg != nil || cap.topic != "" {
		t.Fatalf("expected no publish on delete failure, got topic=%q published=%v", cap.topic, cap.msg != nil)
	}
}

func TestRemoveOrganizationMemberDoesNotPublishWhenDeleteAffectsZeroRows(t *testing.T) {
	db := newTestDB(t)
	if _, err := db.ExecContext(context.Background(), "SET session_replication_role = 'origin'"); err != nil {
		t.Fatalf("enable triggers: %v", err)
	}

	kratosID := "kratos-test-remove-orgmember-zero-rows"
	actorUserID := seedAuthUser(t, db, kratosID)
	targetUserID := seedAuthUser(t, db, "kratos-target-remove-user-zero-rows")
	orgID, roleID, _, _ := seedOrgAndRole(t, db)
	memberID := seedOrganizationMember(t, db, orgID, targetUserID, roleID, actorUserID)

	if _, err := db.ExecContext(context.Background(), `
		CREATE OR REPLACE FUNCTION suppress_org_member_delete() RETURNS trigger AS $$
		BEGIN
			RETURN NULL;
		END;
		$$ LANGUAGE plpgsql;
	`); err != nil {
		t.Fatalf("create suppress trigger function: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `
		CREATE TRIGGER organization_members_delete_suppress
		BEFORE DELETE ON organization_members
		FOR EACH ROW EXECUTE FUNCTION suppress_org_member_delete();
	`); err != nil {
		t.Fatalf("create suppress trigger: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DROP TRIGGER IF EXISTS organization_members_delete_suppress ON organization_members")
		_, _ = db.ExecContext(context.Background(), "DROP FUNCTION IF EXISTS suppress_org_member_delete()")
	})

	cap := &capturePublisher{}
	pub := events.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, pub)

	_, err := svc.RemoveOrganizationMember(authedContext(kratosID), connect.NewRequest(&eventsv1.RemoveOrganizationMemberRequest{
		Id: memberID,
	}))
	if err == nil {
		t.Fatal("expected error when delete affects zero rows")
	}
	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Fatalf("error code = %v, want %v", connectErr.Code(), connect.CodeNotFound)
	}
	if cap.msg != nil || cap.topic != "" {
		t.Fatalf("expected no publish when delete affects zero rows, got topic=%q published=%v", cap.topic, cap.msg != nil)
	}
}

func TestRemoveOrganizationMemberByUserAndOrgPublishesRemovedRole(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-remove-by-user-org"
	actorUserID := seedAuthUser(t, db, kratosID)
	targetUserID := seedAuthUser(t, db, "kratos-target-remove-by-user-org")
	orgID, roleID, orgTitle, roleName := seedOrgAndRole(t, db)
	seedOrganizationMember(t, db, orgID, targetUserID, roleID, actorUserID)

	cap := &capturePublisher{}
	pub := events.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, pub)

	_, err := svc.RemoveOrganizationMemberByUserAndOrg(authedContext(kratosID), connect.NewRequest(&eventsv1.RemoveOrganizationMemberByUserAndOrgRequest{
		OrganizationId: orgID,
		UserId:         targetUserID,
	}))
	if err != nil {
		t.Fatalf("RemoveOrganizationMemberByUserAndOrg() error = %v", err)
	}
	if cap.topic != events.TopicMemberRemoved {
		t.Fatalf("published topic = %q, want %q", cap.topic, events.TopicMemberRemoved)
	}
	if cap.msg == nil {
		t.Fatal("expected a published message, got nil")
	}

	var got events.MemberRemovedPayload
	if err := json.Unmarshal(cap.msg.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got.RoleTemplateID != roleID {
		t.Fatalf("payload RoleTemplateID = %d, want %d", got.RoleTemplateID, roleID)
	}
	if got.RoleName != roleName {
		t.Fatalf("payload RoleName = %q, want %q", got.RoleName, roleName)
	}
	if got.OrganizationTitle != orgTitle {
		t.Fatalf("payload OrganizationTitle = %q, want %q", got.OrganizationTitle, orgTitle)
	}
}

func TestRemoveOrganizationMemberByUserAndOrgDoesNotPublishWhenMemberMissing(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-remove-missing"
	seedAuthUser(t, db, kratosID)
	orgID, _, _, _ := seedOrgAndRole(t, db)

	cap := &capturePublisher{}
	pub := events.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, pub)

	_, err := svc.RemoveOrganizationMemberByUserAndOrg(authedContext(kratosID), connect.NewRequest(&eventsv1.RemoveOrganizationMemberByUserAndOrgRequest{
		OrganizationId: orgID,
		UserId:         999999,
	}))
	if err == nil {
		t.Fatal("expected not found error")
	}
	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Fatalf("error code = %v, want %v", connectErr.Code(), connect.CodeNotFound)
	}
	if cap.msg != nil || cap.topic != "" {
		t.Fatalf("expected no publish when member missing, got topic=%q published=%v", cap.topic, cap.msg != nil)
	}
}

func TestRemoveOrganizationMemberByUserAndOrgDoesNotPublishWhenDeleteAffectsZeroRows(t *testing.T) {
	db := newTestDB(t)
	if _, err := db.ExecContext(context.Background(), "SET session_replication_role = 'origin'"); err != nil {
		t.Fatalf("enable triggers: %v", err)
	}

	kratosID := "kratos-test-remove-zero-rows"
	actorUserID := seedAuthUser(t, db, kratosID)
	targetUserID := seedAuthUser(t, db, "kratos-target-remove-zero-rows")
	orgID, roleID, _, _ := seedOrgAndRole(t, db)
	seedOrganizationMember(t, db, orgID, targetUserID, roleID, actorUserID)

	if _, err := db.ExecContext(context.Background(), `
		CREATE OR REPLACE FUNCTION suppress_org_member_delete() RETURNS trigger AS $$
		BEGIN
			RETURN NULL;
		END;
		$$ LANGUAGE plpgsql;
	`); err != nil {
		t.Fatalf("create suppress trigger function: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `
		CREATE TRIGGER organization_members_delete_suppress
		BEFORE DELETE ON organization_members
		FOR EACH ROW EXECUTE FUNCTION suppress_org_member_delete();
	`); err != nil {
		t.Fatalf("create suppress trigger: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DROP TRIGGER IF EXISTS organization_members_delete_suppress ON organization_members")
		_, _ = db.ExecContext(context.Background(), "DROP FUNCTION IF EXISTS suppress_org_member_delete()")
	})

	cap := &capturePublisher{}
	pub := events.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, pub)

	_, err := svc.RemoveOrganizationMemberByUserAndOrg(authedContext(kratosID), connect.NewRequest(&eventsv1.RemoveOrganizationMemberByUserAndOrgRequest{
		OrganizationId: orgID,
		UserId:         targetUserID,
	}))
	if err == nil {
		t.Fatal("expected error when delete affects zero rows")
	}
	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Fatalf("error code = %v, want %v", connectErr.Code(), connect.CodeNotFound)
	}
	if cap.msg != nil || cap.topic != "" {
		t.Fatalf("expected no publish when delete affects zero rows, got topic=%q published=%v", cap.topic, cap.msg != nil)
	}
}
