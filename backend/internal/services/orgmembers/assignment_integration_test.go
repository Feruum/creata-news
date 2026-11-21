//go:build integration

package orgmembers

import (
	"connectrpc.com/connect"
	"context"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"sync"
	"testing"
)

func TestConcurrentMemberAssignmentsHonorMaxHolders(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "audit-capacity-actor")
	_ = actor
	users := []int32{seedAuthUser(t, db, "audit-capacity-first"), seedAuthUser(t, db, "audit-capacity-second")}
	orgID, roleID, _, _ := seedOrgAndRole(t, db)
	if _, err := db.ExecContext(context.Background(), "UPDATE role_templates SET max_holders=1 WHERE id = ?", roleID); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil)
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, id := range users {
		group.Add(1)
		go func(userID int32) {
			defer group.Done()
			<-start
			_, err := svc.AddOrganizationMember(authedContext("audit-capacity-actor"), connect.NewRequest(&eventsv1.AddOrganizationMemberRequest{OrganizationId: orgID, UserId: userID, RoleTemplateId: roleID}))
			results <- err
		}(id)
	}
	close(start)
	group.Wait()
	close(results)
	success, full := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if connect.CodeOf(err) == connect.CodeResourceExhausted {
			full++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || full != 1 {
		t.Fatalf("assignments: success=%d full=%d", success, full)
	}
}
func TestMembershipInsertRollsBackWhenReconciliationJobCannotPersist(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "audit-outbox-actor")
	_ = actor
	user := seedAuthUser(t, db, "audit-outbox-target")
	orgID, roleID, _, _ := seedOrgAndRole(t, db)
	for _, query := range []string{"SET session_replication_role = 'origin'", `CREATE FUNCTION fail_audit_member_job() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'forced member job failure'; END; $$ LANGUAGE plpgsql`, `CREATE TRIGGER audit_member_job_failure BEFORE INSERT ON domain_event_outbox FOR EACH ROW EXECUTE FUNCTION fail_audit_member_job()`} {
		if _, err := db.ExecContext(context.Background(), query); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil)
	if _, err := svc.AddOrganizationMember(authedContext("audit-outbox-actor"), connect.NewRequest(&eventsv1.AddOrganizationMemberRequest{OrganizationId: orgID, UserId: user, RoleTemplateId: roleID})); err == nil {
		t.Fatal("outbox failure reported as success")
	}
	var count int
	if err := db.NewRaw("SELECT count(*) FROM organization_members WHERE organization_id = ?", orgID).Scan(context.Background(), &count); err != nil || count != 0 {
		t.Fatalf("membership survived outbox failure: %d %v", count, err)
	}
}
