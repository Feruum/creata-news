package news

import (
	"context"
	"testing"
)

func TestAPIPopularUsesRecentViewsAndPublicTranslatedSnapshots(t *testing.T) {
	a := setupAPI(t)
	ctx := context.Background()
	for _, item := range []struct {
		id, kind string
		views    int
		english  string
	}{
		{"low", "publication", 1, "English"},
		{"high", "publication", 3, "English"},
		{"draft", "article", 10, "English"},
		{"untranslated", "publication", 2, ""},
		{"stale", "publication", 0, "English"},
	} {
		article := Doc{"id": item.id, "slug": item.id, "status": "published", "title": Doc{"ru": "Заголовок", "kk": "Атауы", "en": item.english}, "publishedAt": now(), "reviewNotes": []string{"private"}}
		if err := a.store.Write(ctx, func(tx *Tx) error { return tx.Put(ctx, item.kind, article) }); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < item.views; i++ {
			if _, err := a.store.Pool.Exec(ctx, "INSERT INTO creata_events(id,kind,target_id) VALUES($1,'view',$2)", id(), item.id); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := a.store.Pool.Exec(ctx, "INSERT INTO creata_events(id,kind,target_id,created_at) VALUES($1,'view','stale',now()-interval '8 days')", id()); err != nil {
		t.Fatal(err)
	}
	if _, err := a.store.Pool.Exec(ctx, "INSERT INTO creata_events(id,kind,target_id) VALUES($1,'save','low')", id()); err != nil {
		t.Fatal(err)
	}
	response := a.request("GET", "/public/popular?locale=en", nil, 200, "")
	items := array(response["items"])
	if len(items) != 2 || str(object(items[0])["id"]) != "high" || str(object(items[1])["id"]) != "low" {
		t.Fatalf("unexpected popular feed: %v", items)
	}
	if len(array(object(items[0])["reviewNotes"])) != 0 {
		t.Fatal("popular feed exposed private review notes")
	}
}
