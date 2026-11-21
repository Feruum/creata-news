//go:build integration

package achievements

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/studyverse/ems-backend/gen/achievements/v1"
	eventspb "github.com/studyverse/ems-backend/gen/events/v1"
	searchpb "github.com/studyverse/ems-backend/gen/search/v1"
	userspb "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
)

func fixture(t *testing.T) (*bun.DB, *Service, context.Context, int32, int32, int32, int32) {
	t.Helper()
	db := testsupport.ClonedDB(t)
	ctx := context.WithValue(context.Background(), auth.UserIDKey, "achievement-admin")
	var admin, user, org, category int32
	for _, f := range []struct {
		query string
		out   *int32
	}{
		{`INSERT INTO users(kratos_id,username,email) VALUES('achievement-admin','achievement-admin','achievement-admin@test.local') RETURNING id`, &admin},
		{`INSERT INTO users(kratos_id,username,email) VALUES('achievement-user','achievement-user','achievement-user@test.local') RETURNING id`, &user},
		{`INSERT INTO organizations(title,organization_type_id) VALUES('Achievement Club',1) RETURNING id`, &org},
		{`INSERT INTO categories(name,color) VALUES('Achievement Category','#aabbcc') RETURNING id`, &category},
	} {
		if e := db.NewRaw(f.query).Scan(ctx, f.out); e != nil {
			t.Fatal(e)
		}
	}
	if _, err := db.NewRaw("INSERT INTO platform_user_roles(user_id,role) VALUES(?,'administrator'),(?,'none')", admin, user).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	client := testsupport.AllowedPermissions(t)
	return db, NewService(db, authz.NewEnforcer(db, client), nil), ctx, admin, user, org, category
}
func save(t *testing.T, s *Service, ctx context.Context, rule *pb.AchievementRule) *pb.Achievement {
	t.Helper()
	r, e := s.SaveAchievement(ctx, connect.NewRequest(&pb.SaveAchievementRequest{Achievement: &pb.Achievement{Title: "Community Star", Description: "Secret filter description", BadgeIconUrl: "https://example.test/star.png", BackgroundColor: "#112233", AccentColor: "#aabbcc", Effect: pb.AchievementEffect_ACHIEVEMENT_EFFECT_HOLOGRAPHIC, Status: pb.AchievementStatus_ACHIEVEMENT_STATUS_ACTIVE, Rule: rule}}))
	if e != nil {
		t.Fatal(e)
	}
	return r.Msg.Achievement
}
func current(t *testing.T, db *bun.DB, user, id int32) award {
	t.Helper()
	var a award
	if e := db.NewSelect().Model(&a).Where("user_id=? AND achievement_id=?", user, id).Scan(context.Background()); e != nil {
		t.Fatal(e)
	}
	return a
}
func TestAchievementRulesOverridesAndPublicBadges(t *testing.T) {
	db, s, ctx, admin, user, org, category := fixture(t)
	self := context.WithValue(context.Background(), auth.UserIDKey, "achievement-user")
	// Full history already exists before the definition is activated.
	var event, registration int32
	if e := db.NewRaw(`INSERT INTO events(title,description,user_id,organization_id,category_id,structured_location,start_time,end_time,check_in_secret,check_in_token,soc_gpa_badge_status,iros_badge_status) VALUES('Historic event','{}',?,?,?,'{}',NOW()-INTERVAL '2 days',NOW()-INTERVAL '1 day','synthetic-test-secret','testtoken123','approved','pending') RETURNING id`, admin, org, category).Scan(ctx, &event); e != nil {
		t.Fatal(e)
	}
	if e := db.NewRaw(`INSERT INTO event_registrations(event_id,user_id) VALUES(?,?) RETURNING id`, event, user).Scan(ctx, &registration); e != nil {
		t.Fatal(e)
	}
	if _, e := db.NewRaw(`INSERT INTO event_attendance(registration_id,status) VALUES(?,'checked_in')`, registration).Exec(ctx); e != nil {
		t.Fatal(e)
	}
	a := save(t, s, ctx, &pb.AchievementRule{Kind: pb.AchievementRuleKind_ACHIEVEMENT_RULE_KIND_SOC_GPA_ATTENDANCE, Threshold: 1, EventId: &event, OrganizationId: &org, CategoryId: &category})
	if e := Reconcile(ctx, db, a.Id); e != nil {
		t.Fatal(e)
	}
	if !current(t, db, user, a.Id).Automatic {
		t.Fatal("historic approved attendance did not qualify")
	}
	var historyBefore int
	if e := db.NewRaw("SELECT COUNT(*) FROM achievement_history WHERE achievement_id=?", a.Id).Scan(ctx, &historyBefore); e != nil {
		t.Fatal(e)
	}
	if e := Reconcile(ctx, db, a.Id); e != nil {
		t.Fatal(e)
	}
	var historyAfter int
	_ = db.NewRaw("SELECT COUNT(*) FROM achievement_history WHERE achievement_id=?", a.Id).Scan(ctx, &historyAfter)
	if historyBefore != historyAfter {
		t.Fatal("reconciliation must be idempotent")
	}
	if _, e := s.SetSelectedAchievementBadge(self, connect.NewRequest(&pb.SetSelectedAchievementBadgeRequest{AchievementId: a.Id})); e != nil {
		t.Fatal(e)
	}
	projection := &userspb.ListUsersResponse{Users: []*userspb.User{{Id: user}, {Id: admin}}}
	if e := HydrateBadges(ctx, db, projection); e != nil {
		t.Fatal(e)
	}
	if projection.Users[0].SelectedBadge == nil || projection.Users[1].SelectedBadge != nil {
		t.Fatal("batch badge hydration failed")
	}
	nested := &userspb.GetMeResponse{User: &userspb.User{Id: user}}
	if e := HydrateBadges(ctx, db, nested); e != nil || nested.User.SelectedBadge == nil {
		t.Fatalf("nested user badge: %v", e)
	}
	organizers := &eventspb.ListEventOrganizersResponse{Organizers: []*eventspb.EventOrganizer{{UserId: user}}}
	if e := HydrateBadges(ctx, db, organizers); e != nil || organizers.Organizers[0].SelectedBadge == nil {
		t.Fatalf("organizer badge: %v", e)
	}
	members := &eventspb.ListOrganizationMembersResponse{Members: []*eventspb.OrganizationMember{{UserId: user}}}
	if e := HydrateBadges(ctx, db, members); e != nil || members.Members[0].SelectedBadge == nil {
		t.Fatalf("member badge: %v", e)
	}
	historyProjection := &eventspb.GetOrganizationMembershipHistoryResponse{Entries: []*eventspb.MembershipHistoryEntry{{UserId: user, PerformedBy: &user}}}
	if e := HydrateBadges(ctx, db, historyProjection); e != nil || historyProjection.Entries[0].SelectedBadge == nil || historyProjection.Entries[0].PerformedBySelectedBadge == nil {
		t.Fatalf("membership member and actor badges: %v", e)
	}
	searchProjection := &searchpb.GlobalSearchResponse{Results: []*searchpb.SearchResult{{Id: user, Type: searchpb.SearchResultType_SEARCH_RESULT_TYPE_USER}, {Id: user, Type: searchpb.SearchResultType_SEARCH_RESULT_TYPE_EVENT}}}
	if e := HydrateBadges(ctx, db, searchProjection); e != nil || searchProjection.Results[0].SelectedBadge == nil || searchProjection.Results[1].SelectedBadge != nil {
		t.Fatalf("search badge must apply only to user entity: %v", e)
	}
	public, e := s.GetUserAchievement(context.Background(), connect.NewRequest(&pb.GetUserAchievementRequest{UserId: user, AchievementId: a.Id}))
	if e != nil {
		t.Fatal(e)
	}
	if public.Msg.UserAchievement.Achievement.Rule != nil || public.Msg.UserAchievement.Basis != "Confirmed attendance at 1 approved Soc GPA event" || public.Msg.UserAchievement.Manual || public.Msg.UserAchievement.Automatic {
		t.Fatal("public card leaked private rule or award basis")
	}
	if _, e = s.ListAchievements(context.Background(), connect.NewRequest(&pb.ListAchievementsRequest{})); connect.CodeOf(e) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous admin catalog: %v", e)
	}
	// Attendance loss removes automatic state and clears selection, while manual survives.
	if _, e = s.GrantAchievement(ctx, connect.NewRequest(&pb.GrantAchievementRequest{AchievementId: a.Id, UserIds: []int32{user}, Reason: "Community contribution"})); e != nil {
		t.Fatal(e)
	}
	dual := current(t, db, user, a.Id)
	if dual.ManualBasis != "Community contribution" || dual.AutomaticBasis != "Confirmed attendance at 1 approved Soc GPA event" {
		t.Fatalf("manual grant overwrote the automatic basis: %+v", dual)
	}
	if _, e = db.NewRaw("UPDATE event_registrations SET status='cancelled' WHERE id=?", registration).Exec(ctx); e != nil {
		t.Fatal(e)
	}
	if e = Reconcile(ctx, db, a.Id); e != nil {
		t.Fatal(e)
	}
	state := current(t, db, user, a.Id)
	if !state.Manual || state.Automatic {
		t.Fatal("manual award did not survive lost qualification")
	}
	if _, e = s.RevokeAchievement(ctx, connect.NewRequest(&pb.RevokeAchievementRequest{AchievementId: a.Id, UserIds: []int32{user}, Reason: "Correction"})); e != nil {
		t.Fatal(e)
	}
	if _, e = db.NewRaw("UPDATE event_registrations SET status='registered' WHERE id=?", registration).Exec(ctx); e != nil {
		t.Fatal(e)
	}
	var group sync.WaitGroup
	for i := 0; i < 4; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if e := Reconcile(ctx, db, a.Id); e != nil {
				t.Error(e)
			}
		}()
	}
	group.Wait()
	state = current(t, db, user, a.Id)
	if !state.Suppressed || state.Manual || state.Automatic {
		t.Fatal("automatic reconciliation restored an admin revocation")
	}
	if e = HydrateBadges(ctx, db, projection); e != nil {
		t.Fatal(e)
	}
	if projection.Users[0].SelectedBadge != nil {
		t.Fatal("revocation left a selected badge")
	}
	if _, e = s.GrantAchievement(ctx, connect.NewRequest(&pb.GrantAchievementRequest{AchievementId: a.Id, UserIds: []int32{user}})); e != nil {
		t.Fatal(e)
	}
	if e = Reconcile(ctx, db, a.Id); e != nil {
		t.Fatal(e)
	}
	state = current(t, db, user, a.Id)
	if state.Suppressed || !state.Manual || !state.Automatic {
		t.Fatal("explicit grant did not restore qualification")
	}
	if state.AutomaticBasis != "Confirmed attendance at 1 approved Soc GPA event" || state.ManualBasis != "" {
		t.Fatalf("award bases not stored independently after restoration: %+v", state)
	}
	// Revision checks prevent a stale admin form from overwriting a newer rule.
	stale := *a
	a.Rule.Kind = pb.AchievementRuleKind_ACHIEVEMENT_RULE_KIND_IROS_ATTENDANCE
	updated, e := s.SaveAchievement(ctx, connect.NewRequest(&pb.SaveAchievementRequest{Achievement: a}))
	if e != nil {
		t.Fatal(e)
	}
	a = updated.Msg.Achievement
	if _, e = s.SaveAchievement(ctx, connect.NewRequest(&pb.SaveAchievementRequest{Achievement: &stale})); connect.CodeOf(e) != connect.CodeAborted {
		t.Fatalf("stale update should abort: %v", e)
	}
	if e = Reconcile(ctx, db, a.Id); e != nil {
		t.Fatal(e)
	}
	if current(t, db, user, a.Id).Automatic {
		t.Fatal("pending iROS badge qualified")
	}
	if _, e = db.NewRaw("UPDATE events SET iros_badge_status='approved' WHERE id=?", event).Exec(ctx); e != nil {
		t.Fatal(e)
	}
	if e = Reconcile(ctx, db, a.Id); e != nil {
		t.Fatal(e)
	}
	if !current(t, db, user, a.Id).Automatic {
		t.Fatal("approved iROS attendance did not qualify")
	}
	a.Status = pb.AchievementStatus_ACHIEVEMENT_STATUS_ARCHIVED
	if _, e = s.SaveAchievement(ctx, connect.NewRequest(&pb.SaveAchievementRequest{Achievement: a})); e != nil {
		t.Fatal(e)
	}
	if _, e = s.SetSelectedAchievementBadge(self, connect.NewRequest(&pb.SetSelectedAchievementBadgeRequest{AchievementId: a.Id})); e != nil {
		t.Fatal("archived earned badge cannot be selected", e)
	}
	if _, e = db.NewRaw("UPDATE events SET status='cancelled' WHERE id=?", event).Exec(ctx); e != nil {
		t.Fatal(e)
	}
	if e = Reconcile(ctx, db, a.Id); e != nil {
		t.Fatal(e)
	}
	if !current(t, db, user, a.Id).Automatic {
		t.Fatal("archiving failed to preserve owned award")
	}
	if _, e = s.GetUserAchievement(context.Background(), connect.NewRequest(&pb.GetUserAchievementRequest{UserId: user, AchievementId: a.Id})); e != nil {
		t.Fatal("archived earned public badge unavailable", e)
	}
	search := "Secret filter"
	list, e := s.ListAchievements(ctx, connect.NewRequest(&pb.ListAchievementsRequest{Search: &search}))
	if e != nil || list.Msg.Total != 1 {
		t.Fatalf("description search failed: %v %v", list, e)
	}
}

func TestAchievementAuthorizationAndAtomicManualBatches(t *testing.T) {
	db, s, ctx, admin, user, _, _ := fixture(t)
	self := context.WithValue(context.Background(), auth.UserIDKey, "achievement-user")
	a := save(t, s, ctx, nil)
	for name, call := range map[string]func() error{
		"catalog": func() error {
			_, e := s.ListAchievements(self, connect.NewRequest(&pb.ListAchievementsRequest{}))
			return e
		},
		"definition": func() error {
			_, e := s.GetAchievement(self, connect.NewRequest(&pb.GetAchievementRequest{AchievementId: a.Id}))
			return e
		},
		"save": func() error {
			_, e := s.SaveAchievement(self, connect.NewRequest(&pb.SaveAchievementRequest{Achievement: a}))
			return e
		},
		"grant": func() error {
			_, e := s.GrantAchievement(self, connect.NewRequest(&pb.GrantAchievementRequest{AchievementId: a.Id, UserIds: []int32{user}}))
			return e
		},
		"revoke": func() error {
			_, e := s.RevokeAchievement(self, connect.NewRequest(&pb.RevokeAchievementRequest{AchievementId: a.Id, UserIds: []int32{user}, Reason: "correction"}))
			return e
		},
		"journal": func() error {
			_, e := s.ListAchievementHistory(self, connect.NewRequest(&pb.ListAchievementHistoryRequest{}))
			return e
		},
		"recalculate": func() error {
			_, e := s.RecalculateAchievements(self, connect.NewRequest(&pb.RecalculateAchievementsRequest{}))
			return e
		},
		"upload": func() error {
			_, e := s.UploadAchievementImage(self, connect.NewRequest(&pb.UploadAchievementImageRequest{}))
			return e
		},
		"other user's collection": func() error {
			_, e := s.ListUserAchievements(self, connect.NewRequest(&pb.ListUserAchievementsRequest{UserId: admin}))
			return e
		},
	} {
		t.Run(name, func(t *testing.T) {
			if e := call(); connect.CodeOf(e) != connect.CodePermissionDenied {
				t.Fatalf("expected administrator denial, got %v", e)
			}
		})
	}
	if _, e := s.GrantAchievement(ctx, connect.NewRequest(&pb.GrantAchievementRequest{AchievementId: a.Id, UserIds: []int32{user, 2147483647}})); connect.CodeOf(e) != connect.CodeInvalidArgument {
		t.Fatalf("invalid batch: %v", e)
	}
	var count int
	if e := db.NewRaw("SELECT COUNT(*) FROM user_achievements WHERE achievement_id=?", a.Id).Scan(ctx, &count); e != nil || count != 0 {
		t.Fatalf("invalid batch partially applied: %d %v", count, e)
	}
	if _, e := s.RevokeAchievement(ctx, connect.NewRequest(&pb.RevokeAchievementRequest{AchievementId: a.Id, UserIds: []int32{user}})); connect.CodeOf(e) != connect.CodeInvalidArgument {
		t.Fatalf("empty revoke reason: %v", e)
	}
	grant, e := s.GrantAchievement(ctx, connect.NewRequest(&pb.GrantAchievementRequest{AchievementId: a.Id, UserIds: []int32{user, user}, Reason: "Contribution"}))
	if e != nil || grant.Msg.Affected != 1 {
		t.Fatalf("batch should deduplicate users: %v %v", grant, e)
	}
	collection, e := s.ListUserAchievements(self, connect.NewRequest(&pb.ListUserAchievementsRequest{AchievementId: a.Id}))
	if e != nil || len(collection.Msg.Achievements) != 1 || !collection.Msg.Achievements[0].Manual || collection.Msg.Achievements[0].Basis != "Contribution" {
		t.Fatalf("private owner collection: %v %v", collection, e)
	}
	if _, e := s.SetSelectedAchievementBadge(ctx, connect.NewRequest(&pb.SetSelectedAchievementBadgeRequest{AchievementId: a.Id})); connect.CodeOf(e) != connect.CodeNotFound {
		t.Fatalf("admin selected another user's award: %v", e)
	}
}

func TestDurableQueueRollbackRetriesAndRuleFilters(t *testing.T) {
	db, s, ctx, admin, user, org, category := fixture(t)
	var event, reg int32
	if e := db.NewRaw(`INSERT INTO events(title,description,user_id,organization_id,category_id,structured_location,start_time,end_time,check_in_secret,check_in_token) VALUES('Filter event','{}',?,?,?,'{}',NOW(),NOW()+INTERVAL '1 hour','synthetic-test-secret','testtoken123') RETURNING id`, admin, org, category).Scan(ctx, &event); e != nil {
		t.Fatal(e)
	}
	if e := db.NewRaw(`INSERT INTO event_registrations(event_id,user_id) VALUES(?,?) RETURNING id`, event, user).Scan(ctx, &reg); e != nil {
		t.Fatal(e)
	}
	a := save(t, s, ctx, &pb.AchievementRule{Kind: pb.AchievementRuleKind_ACHIEVEMENT_RULE_KIND_REGISTRATIONS, Threshold: 1, EventId: &event, OrganizationId: &org, CategoryId: &category})
	if _, e := db.NewRaw("DELETE FROM achievement_recalculation_jobs").Exec(ctx); e != nil {
		t.Fatal(e)
	}
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.NewRaw("UPDATE event_registrations SET status='cancelled' WHERE id=?", reg).Exec(ctx); e != nil {
		t.Fatal(e)
	}
	_ = tx.Rollback()
	var count int
	if e = db.NewRaw("SELECT COUNT(*) FROM achievement_recalculation_jobs").Scan(ctx, &count); e != nil {
		t.Fatal(e)
	}
	if count != 0 {
		t.Fatal("rolled back source mutation leaked a job")
	}
	if _, e = db.NewRaw("UPDATE event_registrations SET status='registered' WHERE id=?", reg).Exec(ctx); e != nil {
		t.Fatal(e)
	}
	if e = NewWorker(db).ProcessNext(ctx); e != nil {
		t.Fatal(e)
	}
	if e = NewWorker(db).ProcessNext(ctx); e != nil {
		t.Fatal(e)
	}
	if !current(t, db, user, a.Id).Automatic {
		t.Fatal("durably queued registration did not award")
	}
	// All filters are ANDed. A different category removes the automatic award.
	var other int32
	if e = db.NewRaw("INSERT INTO categories(name,color) VALUES('Other','#ffffff') RETURNING id").Scan(ctx, &other); e != nil {
		t.Fatal(e)
	}
	if _, e = db.NewRaw("UPDATE events SET category_id=? WHERE id=?", other, event).Exec(ctx); e != nil {
		t.Fatal(e)
	}
	if e = NewWorker(db).ProcessNext(ctx); e != nil {
		t.Fatal(e)
	}
	if e = NewWorker(db).ProcessNext(ctx); e != nil {
		t.Fatal(e)
	}
	if current(t, db, user, a.Id).Automatic {
		t.Fatal("category AND filter ignored")
	}
	// A corrupt stored rule produces a retained retry job, never silently drops it.
	if _, e = db.NewRaw("UPDATE achievements SET rule=? WHERE id=?", json.RawMessage(`{"kind":99,"threshold":1}`), a.Id).Exec(ctx); e != nil {
		t.Fatal(e)
	}
	later := save(t, s, ctx, &pb.AchievementRule{Kind: pb.AchievementRuleKind_ACHIEVEMENT_RULE_KIND_REGISTRATIONS, Threshold: 1})
	if _, e = db.NewRaw("DELETE FROM achievement_recalculation_jobs").Exec(ctx); e != nil {
		t.Fatal(e)
	}
	if e = enqueue(ctx, db, 0); e != nil {
		t.Fatal(e)
	}
	if e = NewWorker(db).ProcessNext(ctx); e != nil {
		t.Fatalf("global job must fan out definitions: %v", e)
	}
	if e = NewWorker(db).ProcessNext(ctx); e == nil {
		t.Fatal("invalid rule should fail reconciliation")
	}
	var job struct {
		Attempts  int
		LastError string
	}
	if e = db.NewRaw("SELECT attempts,last_error FROM achievement_recalculation_jobs WHERE achievement_id=?", a.Id).Scan(ctx, &job); e != nil {
		t.Fatal(e)
	}
	if job.Attempts < 1 || job.LastError == "" {
		t.Fatal("failed job lacks durable retry details")
	}
	if e = NewWorker(db).ProcessNext(ctx); e != nil {
		t.Fatalf("bad first definition starved later definition: %v", e)
	}
	if !current(t, db, user, later.Id).Automatic {
		t.Fatal("later definition was not processed independently")
	}
}

func TestAchievementEnumMigrationPreservesExistingDefinitions(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	if _, e := db.ExecContext(ctx, `CREATE TABLE achievements(id integer PRIMARY KEY,status integer NOT NULL DEFAULT 0,effect integer NOT NULL DEFAULT 0,
		CONSTRAINT achievements_status_valid CHECK(status IN (0,1,2)),CONSTRAINT achievements_effect_valid CHECK(effect IN (0,1)));
		INSERT INTO achievements(id,status,effect) VALUES(1,0,0),(2,1,1),(3,2,0)`); e != nil {
		t.Fatal(e)
	}
	migration, e := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "packages", "db", "migrations", "0037_achievement-enum-values.sql"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, string(migration)); e != nil {
		t.Fatal(e)
	}
	var rows []struct {
		ID     int
		Status int
		Effect int
	}
	if e = db.NewRaw("SELECT id,status,effect FROM achievements ORDER BY id").Scan(ctx, &rows); e != nil {
		t.Fatal(e)
	}
	for i, row := range rows {
		expectedEffect := 1
		if i == 1 {
			expectedEffect = 2
		}
		if row.Status != i+1 || row.Effect != expectedEffect {
			t.Fatalf("enum ordinals changed meaning: %+v", rows)
		}
	}
}
