package news

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
)

func TestConcurrentEditorMergesHaveOneWinner(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	if err := a.store.Write(ctx, func(tx *Tx) error {
		for _, sid := range []string{"race-left", "race-right"} {
			if err := tx.Put(ctx, "story", Doc{"id": sid, "status": "new", "updatedAt": "initial"}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(Doc{"otherId": "race-right", "expectedUpdatedAt": "initial", "otherExpectedUpdatedAt": "initial"})
	statuses := make(chan int, 2)
	var group sync.WaitGroup
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			request, _ := http.NewRequest("POST", a.server.URL+base+"/admin/bank/race-left/merge", bytes.NewReader(payload))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-CSRF-Token", a.csrf)
			response, err := a.client.Do(request)
			if err != nil {
				statuses <- 0
				return
			}
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			statuses <- response.StatusCode
		}()
	}
	group.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatalf("merge race produced %v", counts)
	}
	merges, _ := a.store.List(ctx, "story_merge")
	if len(merges) != 1 {
		t.Fatal("race created multiple merge histories")
	}
}

func TestNearStoryCandidatesRequireEditorMergeAndUndoPreservesSources(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	left := Doc{"id": "left", "title": "Orbital laboratory launches reusable satellite platform for science teams", "text": "A laboratory announced a reusable satellite platform for science teams today.", "locale": "en", "status": "new", "sources": []any{Doc{"url": "https://example.org/one"}}, "evidence": []any{Doc{"url": "https://example.org/one", "text": "One"}}, "articleId": "article-one", "updatedAt": "2026-11-14T00:00:00Z"}
	right := Doc{"id": "right", "title": "Orbital laboratory launches new reusable satellite platform for science teams", "text": "The laboratory announced a reusable satellite platform for science teams this morning.", "locale": "en", "status": "classified", "sources": []any{Doc{"url": "https://example.org/two"}}, "evidence": []any{Doc{"url": "https://example.org/two", "text": "Two"}}, "articleId": "article-two", "updatedAt": "2026-11-14T00:00:00Z"}
	err := a.store.Write(ctx, func(tx *Tx) error {
		for _, d := range []Doc{left, right} {
			if e := tx.Put(ctx, "story", d); e != nil {
				return e
			}
		}
		if e := tx.Put(ctx, "raw_item", Doc{"id": "raw-right", "storyId": "right", "text": "Raw remains intact"}); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, "INSERT INTO creata_fingerprints(fingerprint,story_id) VALUES('fp-right','right')")
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	candidates := array(a.request("GET", "/admin/bank/similar", nil, 200, "")["items"])
	if len(candidates) != 1 || len(array(object(candidates[0])["reasons"])) == 0 {
		t.Fatalf("candidate missing: %v", candidates)
	}
	bank := array(a.request("GET", "/admin/bank", nil, 200, "")["items"])
	if len(bank) != 2 {
		t.Fatal("candidate silently merged")
	}
	input := Doc{"otherId": "right", "expectedUpdatedAt": left["updatedAt"], "otherExpectedUpdatedAt": right["updatedAt"]}
	merged := a.request("POST", "/admin/bank/left/merge", input, 200, "")
	mid := str(merged["id"])
	a.request("POST", "/admin/bank/left/merge", input, 409, "")
	survivor, _ := a.store.Get(ctx, "story", "left")
	if len(array(survivor["sources"])) != 2 || !has(survivor["linkedArticleIds"], "article-two") {
		t.Fatal("merge lost sources or articles")
	}
	raw, _ := a.store.Get(ctx, "raw_item", "raw-right")
	if str(raw["text"]) != "Raw remains intact" {
		t.Fatal("raw source changed")
	}
	a.request("POST", "/admin/bank/merges/"+mid+"/undo", nil, 200, "")
	restored, _ := a.store.Get(ctx, "story", "right")
	if str(restored["status"]) != "classified" || str(restored["articleId"]) != "article-two" {
		t.Fatal("undo did not restore original story")
	}
	var target string
	if e := a.store.Pool.QueryRow(ctx, "SELECT story_id FROM creata_fingerprints WHERE fingerprint='fp-right'").Scan(&target); e != nil || target != "right" {
		t.Fatal("undo lost fingerprint ownership")
	}
	a.request("POST", "/admin/bank/merges/"+mid+"/undo", nil, 409, "")
}

func TestMergeUndoRefusesToEraseInterveningCollection(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	if e := a.store.Write(ctx, func(tx *Tx) error {
		for _, sid := range []string{"a", "b"} {
			if e := tx.Put(ctx, "story", Doc{"id": sid, "title": sid, "status": "new", "updatedAt": "v1"}); e != nil {
				return e
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	m := a.request("POST", "/admin/bank/a/merge", Doc{"otherId": "b", "expectedUpdatedAt": "v1", "otherExpectedUpdatedAt": "v1"}, 200, "")
	if e := a.store.Write(ctx, func(tx *Tx) error {
		d, _ := tx.Get(ctx, "story", "a")
		d["text"] = "New evidence"
		return tx.Put(ctx, "story", d)
	}); e != nil {
		t.Fatal(e)
	}
	a.request("POST", "/admin/bank/merges/"+str(m["id"])+"/undo", nil, 409, "")
	kept, _ := a.store.Get(ctx, "story", "a")
	if str(kept["text"]) != "New evidence" {
		t.Fatal("undo erased new evidence")
	}
	leftEvidence := a.request("GET", "/admin/bank/a/evidence", nil, 200, "")
	rightEvidence := a.request("GET", "/admin/bank/b/evidence", nil, 200, "")
	a.request("POST", "/admin/bank/merges/"+str(m["id"])+"/split", Doc{"expectedLeftVersion": "stale", "expectedRightVersion": rightEvidence["version"]}, 409, "")
	a.request("POST", "/admin/bank/merges/"+str(m["id"])+"/split", Doc{"expectedLeftVersion": leftEvidence["version"], "expectedRightVersion": rightEvidence["version"]}, 200, "")
	kept, _ = a.store.Get(ctx, "story", "a")
	if str(kept["text"]) != "New evidence" {
		t.Fatal("split erased new evidence")
	}
	right, _ := a.store.Get(ctx, "story", "b")
	if str(right["mergedInto"]) != "" {
		t.Fatal("split did not restore donor story")
	}
}
