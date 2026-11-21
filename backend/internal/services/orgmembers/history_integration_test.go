//go:build integration

package orgmembers

import (
	"context"
	"errors"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
	"testing"
)

func TestBulkMembershipHistoryParticipatesInTransactionAndPropagatesErrors(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	actor := seedAuthUser(t, db, "audit-history")
	orgID, roleID, orgTitle, _ := seedOrgAndRole(t, db)
	seedOrganizationMember(t, db, orgID, actor, roleID, actor)
	rollback := errors.New("abort organization change")
	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := RecordBulkMembershipHistoryInTransaction(ctx, tx, orgID, orgTitle, models.MembershipHistoryActionOrgArchived, &actor); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	var count int
	if err := db.NewRaw("SELECT count(*) FROM membership_history").Scan(ctx, &count); err != nil || count != 0 {
		t.Fatalf("history persisted after rollback: %d %v", count, err)
	}
	if _, err := db.ExecContext(ctx, "SET session_replication_role = 'origin'"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE FUNCTION fail_audit_history() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'forced history failure'; END; $$ LANGUAGE plpgsql`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER audit_history_failure BEFORE INSERT ON membership_history FOR EACH ROW EXECUTE FUNCTION fail_audit_history()`); err != nil {
		t.Fatal(err)
	}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return RecordBulkMembershipHistoryInTransaction(ctx, tx, orgID, orgTitle, models.MembershipHistoryActionOrgArchived, &actor)
	})
	if err == nil {
		t.Fatal("history insert error ignored")
	}
}
