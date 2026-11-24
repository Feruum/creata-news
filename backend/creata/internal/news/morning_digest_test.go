package news

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestMorningDigestStaysDraftAcrossRestart(t *testing.T) {
	a := setupAPI(t)
	a.app.Config.TelegramChannelOnly = true
	a.login("chief_editor")
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	seedMorningDigest(t, a.store, today)

	if err := a.app.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	initial := array(a.request("GET", "/admin/digests", nil, 200, "")["items"])
	if len(initial) != 1 {
		t.Fatalf("expected one morning draft, got %d", len(initial))
	}
	draft := object(initial[0])
	if str(draft["status"]) != "draft" || str(draft["kind"]) != "channel" || str(draft["morningDate"]) != today.Format(time.DateOnly) {
		t.Fatalf("unexpected morning draft: %v", draft)
	}
	if ids := digestItemIDs(draft, "articles"); !slices.Equal(ids, []string{"kazakhstan", "uzbekistan"}) {
		t.Fatalf("wrong articles: %v", ids)
	}
	if ids := digestItemIDs(draft, "opportunities"); !slices.Equal(ids, []string{"regional-grant"}) {
		t.Fatalf("wrong opportunities: %v", ids)
	}
	if err := a.store.Write(ctx, func(tx *Tx) error {
		return tx.Put(ctx, "job", Doc{"id": "confirmed-collection", "kind": "collect", "status": "succeeded"})
	}); err != nil {
		t.Fatal(err)
	}
	a.login("author")
	summary := a.request("GET", "/admin/digests/morning", nil, 200, "")
	edition := object(summary["edition"])
	if str(edition["id"]) != str(draft["id"]) || integer(edition["articleCount"]) != 2 || integer(edition["opportunityCount"]) != 1 || parseTime(summary["lastCollectionAt"]).IsZero() {
		t.Fatalf("wrong morning overview: %v", summary)
	}

	reopened, err := Open(ctx, a.store.Pool.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Pool.Close)
	restarted, err := New(reopened, a.app.Config)
	if err != nil {
		t.Fatal(err)
	}
	completed := make(chan error, 2)
	go func() { completed <- a.app.Tick(ctx) }()
	go func() { completed <- restarted.Tick(ctx) }()
	for range 2 {
		if err = <-completed; err != nil {
			t.Fatal(err)
		}
	}
	afterRestart, err := reopened.List(ctx, "digest")
	if err != nil {
		t.Fatal(err)
	}
	if len(afterRestart) != 1 || !reflect.DeepEqual(draft, afterRestart[0]) {
		t.Fatal("restart changed the existing draft or created a duplicate")
	}
	deliveries, err := reopened.List(ctx, "delivery")
	if err != nil || len(deliveries) != 0 {
		t.Fatalf("morning draft created a delivery: %v, error %v", deliveries, err)
	}
}

func TestMorningDigestUsesLocalCalendarAndCreatesAnEmptyDraft(t *testing.T) {
	a := setupAPI(t)
	ctx := context.Background()
	location, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		t.Fatal(err)
	}
	firstMorning := time.Date(2026, time.November, 24, 9, 0, 0, 0, location)
	tickMorning := func(at time.Time) []Doc {
		t.Helper()
		if err := a.store.Write(ctx, func(tx *Tx) error {
			return a.app.scheduleMorningDigest(tx, ctx, at, Doc{})
		}); err != nil {
			t.Fatal(err)
		}
		editions, err := a.store.List(ctx, "digest")
		if err != nil {
			t.Fatal(err)
		}
		return editions
	}
	if before := tickMorning(firstMorning.Add(-time.Second)); len(before) != 0 {
		t.Fatal("edition created before 09:00 in the newsroom time zone")
	}
	first := tickMorning(firstMorning)
	if len(first) != 1 || str(first[0]["morningDate"]) != "2026-11-24" || str(first[0]["status"]) != "draft" || len(array(first[0]["articles"])) != 0 {
		t.Fatalf("expected an empty November 24 draft: %v", first)
	}
	second := tickMorning(firstMorning.AddDate(0, 0, 1).Add(time.Hour))
	if len(second) != 2 {
		t.Fatalf("expected another draft on the next local day: %v", second)
	}
}

func TestManualChannelDigestUsesCentralAsia(t *testing.T) {
	a := setupAPI(t)
	a.login("editor")
	seedMorningDigest(t, a.store, time.Now().UTC().Truncate(24*time.Hour))

	draft := a.request("POST", "/admin/digests/preview", Doc{"locale": "kk"}, 201, "")
	if ids := digestItemIDs(draft, "articles"); slices.Contains(ids, "tokyo") || slices.Contains(ids, "unpublished") || !slices.Contains(ids, "kazakhstan") || !slices.Contains(ids, "uzbekistan") {
		t.Fatalf("wrong articles: %v", ids)
	}
	if ids := digestItemIDs(draft, "opportunities"); !slices.Equal(ids, []string{"regional-grant"}) {
		t.Fatalf("wrong opportunities: %v", ids)
	}
}

func seedMorningDigest(t *testing.T, store *Store, today time.Time) {
	t.Helper()
	err := store.Write(context.Background(), func(tx *Tx) error {
		ctx := context.Background()
		settings := DefaultSettings()
		settings["morningDigestHour"], settings["timezone"] = 0, "UTC"
		if err := tx.Put(ctx, "settings", settings); err != nil {
			return err
		}
		for _, story := range []struct {
			id, region string
			ago        time.Duration
		}{
			{"kazakhstan", " Kazakhstan ", time.Hour},
			{"uzbekistan", "UZ", 2 * time.Hour},
			{"tokyo", "JP", 30 * time.Minute},
			{"older", "KG", 25 * time.Hour},
		} {
			publication := Doc{"id": story.id, "region": story.region, "slug": story.id, "title": localized(story.id), "publishedAt": today.Add(-story.ago).Format(time.RFC3339)}
			if err := tx.Put(ctx, "publication", publication); err != nil {
				return err
			}
		}
		if err := tx.Put(ctx, "article", Doc{"id": "unpublished", "region": "KZ"}); err != nil {
			return err
		}
		for _, opportunity := range []struct{ id, country, status string }{
			{"regional-grant", "Қырғызстан", "approved"},
			{"global-grant", "Worldwide", "approved"},
			{"unapproved-grant", "TM", "pending"},
		} {
			entry := Doc{"id": opportunity.id, "country": opportunity.country, "status": opportunity.status, "deadline": today.Add(48 * time.Hour).Format(time.RFC3339)}
			if err := tx.Put(ctx, "opportunity", entry); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func digestItemIDs(digest Doc, field string) []string {
	ids := []string{}
	for _, item := range array(digest[field]) {
		ids = append(ids, str(object(item)["id"]))
	}
	return ids
}
