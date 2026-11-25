package news

import (
	"context"
	"testing"
	"time"
)

func TestLegacyArticleStatusMigrationIsConservativeAndIdempotent(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	create := func() string {
		return str(a.request("POST", "/admin/articles", validArticle(), 201, "")["id"])
	}
	act := func(aid, action string) {
		a.request("POST", "/admin/articles/"+aid+"/actions", Doc{
			"action": action, "expectedRevision": 1,
			"scheduledAt": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		}, 200, "")
	}

	scheduled, withdrawn, reviewing, publishedScheduled := create(), create(), create(), create()
	act(scheduled, "approve")
	act(scheduled, "schedule")
	act(publishedScheduled, "approve")
	act(publishedScheduled, "publish")
	for _, aid := range []string{withdrawn, reviewing} {
		act(aid, "approve")
		act(aid, "publish")
		act(aid, "unpublish")
	}
	act(reviewing, "submit") // a newer editorial decision must not be overwritten

	// Recreate the statuses stored by the previous release, then rerun schema
	// initialization as an upgrade from migration 1.
	if e := a.store.Write(ctx, func(tx *Tx) error {
		for aid, status := range map[string]string{scheduled: "approved", withdrawn: "review", publishedScheduled: "published"} {
			article, err := tx.Get(ctx, "article", aid)
			if err != nil {
				return err
			}
			article["status"] = status
			if aid == publishedScheduled {
				article["scheduledAt"] = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
			}
			if err = tx.Put(ctx, "article", article); err != nil {
				return err
			}
		}
		if _, e := tx.Exec(ctx, "DELETE FROM creata_migrations WHERE version=2"); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, schema)
		return e
	}); e != nil {
		t.Fatal(e)
	}

	for aid, status := range map[string]string{scheduled: "scheduled", withdrawn: "unpublished", reviewing: "review", publishedScheduled: "scheduled"} {
		article, e := a.store.Get(ctx, "article", aid)
		if e != nil || str(article["status"]) != status {
			t.Fatalf("wanted %s, got %v (%v)", status, article, e)
		}
	}
	if _, err := a.store.Get(ctx, "publication", publishedScheduled); err != nil {
		t.Fatal("legacy scheduled update lost the live snapshot", err)
	}
	// The next startup cannot undo a deliberate review action after migration.
	act(withdrawn, "submit")
	if e := a.store.Write(ctx, func(tx *Tx) error { _, e := tx.Exec(ctx, schema); return e }); e != nil {
		t.Fatal(e)
	}
	article, e := a.store.Get(ctx, "article", withdrawn)
	if e != nil || str(article["status"]) != "review" {
		t.Fatal("migration was not idempotent", article, e)
	}
}

func TestScheduledArticleCancellationAndWithdrawalGuard(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	aid := str(a.request("POST", "/admin/articles", validArticle(), 201, "")["id"])
	path := "/admin/articles/" + aid + "/actions"
	act := func(action string, code int) Doc {
		return a.request("POST", path, Doc{"action": action, "expectedRevision": 1, "scheduledAt": time.Now().Add(time.Hour).UTC().Format(time.RFC3339), "note": "Editorial change"}, code, "")
	}
	act("unpublish", 409) // a never-published material cannot be withdrawn
	act("approve", 200)
	act("schedule", 200)
	act("approve", 409) // accidental duplicate approval must not cancel an active schedule
	current := a.request("GET", "/admin/articles/"+aid, nil, 200, "")
	if str(current["status"]) != "scheduled" || str(current["scheduledAt"]) == "" {
		t.Fatal(current)
	}
	submitted := act("submit", 200)
	if str(submitted["status"]) != "review" || submitted["scheduledAt"] != nil || submitted["approvedRevision"] != nil {
		t.Fatal(submitted)
	}
	act("approve", 200)
	act("schedule", 200)
	reworked := act("rework", 200)
	if str(reworked["status"]) != "rework" || reworked["scheduledAt"] != nil || reworked["approvedRevision"] != nil {
		t.Fatal(reworked)
	}
}

func TestPostizScheduledTargetsRemainWhenTimingChanges(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	attachPostiz(t, a, "now")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	postizAction(a, article, "approve", nil, 200)
	schedule := selections("one", "two")
	schedule["scheduledAt"] = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	postizAction(a, article, "schedule", schedule, 200)
	postizAction(a, article, "schedule", Doc{"scheduledAt": time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)}, 200)
	if postizItems(a, article)["one/en"]["status"] != "scheduled" || postizItems(a, article)["two/en"]["status"] != "scheduled" {
		t.Fatal("changing timing cancelled selected deliveries")
	}
	postizAction(a, article, "publish", nil, 200)
	if postizItems(a, article)["one/en"]["status"] != "queued" || postizItems(a, article)["two/en"]["status"] != "queued" {
		t.Fatal("publishing immediately cancelled selected deliveries")
	}
}
