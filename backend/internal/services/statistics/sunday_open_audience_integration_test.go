//go:build integration

package statistics

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/encoding/protojson"
)

func audienceFixture(t *testing.T) (*bun.DB, *Service, context.Context, context.Context) {
	t.Helper()
	db := testsupport.IsolatedDB(t)
	for _, table := range []string{"users", "platform_user_roles", "sunday_open_accounts", "sunday_open_onboarding"} {
		if _, err := db.ExecContext(context.Background(), "CREATE TABLE "+table+" (LIKE public."+table+" INCLUDING ALL)"); err != nil {
			t.Fatal(err)
		}
	}
	for _, actor := range []struct{ name, role string }{{"audience-staff", "staff"}, {"audience-regular", "none"}} {
		var id int32
		if err := db.NewRaw(`INSERT INTO users(kratos_id,username,email,password) VALUES(?,?,?,'test') RETURNING id`, actor.name, actor.name, actor.name+"@test.local").Scan(context.Background(), &id); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(context.Background(), `INSERT INTO platform_user_roles(user_id,role) VALUES(?,?)`, id, actor.role); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(db, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)))
	return db, service, context.WithValue(context.Background(), auth.UserIDKey, "audience-staff"), context.WithValue(context.Background(), auth.UserIDKey, "audience-regular")
}

func audienceCounts(buckets []*eventsv1.SundayOpenAudienceBucket) map[string]int32 {
	counts := map[string]int32{}
	for _, bucket := range buckets {
		counts[bucket.Key] = bucket.Count
	}
	return counts
}

func TestSundayOpenAudienceCountsCurrentCompletedProfilesWithoutMultiselectMultiplication(t *testing.T) {
	db, service, staffCtx, _ := audienceFixture(t)
	ids := make([]string, 4)
	for i := range ids {
		if err := db.NewRaw(`INSERT INTO sunday_open_accounts(full_name) VALUES('Audience visitor') RETURNING id`).Scan(staffCtx, &ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(staffCtx, `INSERT INTO sunday_open_onboarding(account_id,role,interests,goals,formats,source,source_other,completed_at) VALUES
		(?,'student',ARRAY['it_ai','design'],ARRAY['try_new','meet_people'],ARRAY['talks','workshops'],'friends','',NOW()),
		(?,'parent',ARRAY['design'],ARRAY['meet_people'],ARRAY['talks'],'other','Private source text',NOW());
		INSERT INTO sunday_open_onboarding(account_id,prompted_at) VALUES(?,NOW())`, ids[0], ids[1], ids[2]); err != nil {
		t.Fatal(err)
	}
	response, err := service.GetSundayOpenAudienceStatistics(staffCtx, connect.NewRequest(&eventsv1.GetSundayOpenAudienceStatisticsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.TotalAccounts != 4 || response.Msg.CompletedAccounts != 2 {
		t.Fatalf("incorrect audience totals: %v", response.Msg)
	}
	for name, counts := range map[string]struct{ got, want map[string]int32 }{
		"roles":     {audienceCounts(response.Msg.Roles), map[string]int32{"student": 1, "parent": 1}},
		"interests": {audienceCounts(response.Msg.Interests), map[string]int32{"it_ai": 1, "design": 2}},
		"goals":     {audienceCounts(response.Msg.Goals), map[string]int32{"try_new": 1, "meet_people": 2}},
		"formats":   {audienceCounts(response.Msg.Formats), map[string]int32{"talks": 2, "workshops": 1}},
		"sources":   {audienceCounts(response.Msg.Sources), map[string]int32{"friends": 1, "other": 1}},
	} {
		if len(counts.got) != len(counts.want) {
			t.Errorf("%s buckets got %v want %v", name, counts.got, counts.want)
		}
		for key, count := range counts.want {
			if counts.got[key] != count {
				t.Errorf("%s/%s got %d want %d", name, key, counts.got[key], count)
			}
		}
	}
	encoded := protojson.Format(response.Msg)
	if strings.Contains(encoded, "Private source") || strings.Contains(encoded, ids[0]) {
		t.Fatal("aggregate response exposed individual answers")
	}
	if _, err := db.ExecContext(staffCtx, `UPDATE sunday_open_onboarding SET role='alumni',interests=ARRAY['sports_games'],goals=ARRAY['find_club'],formats=ARRAY['competitions'],source='search',updated_at=NOW() WHERE account_id=?`, ids[0]); err != nil {
		t.Fatal(err)
	}
	response, err = service.GetSundayOpenAudienceStatistics(staffCtx, connect.NewRequest(&eventsv1.GetSundayOpenAudienceStatisticsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.CompletedAccounts != 2 || audienceCounts(response.Msg.Roles)["student"] != 0 || audienceCounts(response.Msg.Roles)["alumni"] != 1 || audienceCounts(response.Msg.Interests)["design"] != 1 || audienceCounts(response.Msg.Interests)["sports_games"] != 1 {
		t.Fatalf("editing double counted profile: %v", response.Msg)
	}
}

func TestSundayOpenAudienceEmptyAndPromptOnlyProfiles(t *testing.T) {
	db, service, staffCtx, _ := audienceFixture(t)
	request := connect.NewRequest(&eventsv1.GetSundayOpenAudienceStatisticsRequest{})
	response, err := service.GetSundayOpenAudienceStatistics(staffCtx, request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.TotalAccounts != 0 || response.Msg.CompletedAccounts != 0 || len(response.Msg.Roles)+len(response.Msg.Interests)+len(response.Msg.Goals)+len(response.Msg.Formats)+len(response.Msg.Sources) != 0 {
		t.Fatalf("empty audience: %v", response.Msg)
	}
	if _, err := db.ExecContext(staffCtx, `WITH account AS (INSERT INTO sunday_open_accounts(full_name) VALUES('Prompted visitor') RETURNING id) INSERT INTO sunday_open_onboarding(account_id,prompted_at) SELECT id,NOW() FROM account`); err != nil {
		t.Fatal(err)
	}
	response, err = service.GetSundayOpenAudienceStatistics(staffCtx, request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.TotalAccounts != 1 || response.Msg.CompletedAccounts != 0 || len(response.Msg.Roles)+len(response.Msg.Interests)+len(response.Msg.Goals)+len(response.Msg.Formats)+len(response.Msg.Sources) != 0 {
		t.Fatalf("prompted visitor counted as completed: %v", response.Msg)
	}
}

func TestSundayOpenAudienceRequiresCurrentStaffBeforeAudienceDatabaseAccess(t *testing.T) {
	db, service, staffCtx, regularCtx := audienceFixture(t)
	if _, err := db.ExecContext(staffCtx, `ALTER TABLE sunday_open_onboarding DROP COLUMN role`); err != nil {
		t.Fatal(err)
	}
	request := connect.NewRequest(&eventsv1.GetSundayOpenAudienceStatisticsRequest{})
	if _, err := service.GetSundayOpenAudienceStatistics(regularCtx, request); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("nonstaff request reached audience storage: %v", err)
	}
	if _, err := service.GetSundayOpenAudienceStatistics(staffCtx, request); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("staff storage failure was masked: %v", err)
	}
}
