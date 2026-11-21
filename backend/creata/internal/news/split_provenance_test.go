package news

import (
	"context"
	"testing"
)

func TestSplitRestoresPostMergeDonorRawAndTextFingerprintOwnership(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	src := Doc{"id": "split-source", "type": "rss"}
	first := Doc{"url": "https://example.org/a", "title": "Story A", "text": "Original source A", "locale": "en"}
	donor := Doc{"url": "https://example.org/b", "title": "Story B", "text": "Original donor text B1", "locale": "en"}
	ingest := func(item Doc) {
		t.Helper()
		if e := a.store.Write(ctx, func(tx *Tx) error { return a.app.ingest(tx, ctx, src, item) }); e != nil {
			t.Fatal(e)
		}
	}
	ingest(first)
	ingest(donor)
	stories, _ := a.store.List(ctx, "story")
	var left, right Doc
	for _, story := range stories {
		if str(story["url"]) == str(first["url"]) {
			left = story
		} else {
			right = story
		}
	}
	merge := a.request("POST", "/admin/bank/"+str(left["id"])+"/merge", Doc{"otherId": right["id"], "expectedUpdatedAt": left["updatedAt"], "otherExpectedUpdatedAt": right["updatedAt"]}, 200, "")
	donor["text"] = "Updated donor text B2 with a distinct measured finding"
	ingest(donor)
	le := a.request("GET", "/admin/bank/"+str(left["id"])+"/evidence", nil, 200, "")
	re := a.request("GET", "/admin/bank/"+str(right["id"])+"/evidence", nil, 200, "")
	a.request("POST", "/admin/bank/merges/"+str(merge["id"])+"/split", Doc{"expectedLeftVersion": le["version"], "expectedRightVersion": re["version"]}, 200, "")
	fingerprint := hash("text:" + str(donor["text"]))
	var owner string
	if e := a.store.Pool.QueryRow(ctx, "SELECT story_id FROM creata_fingerprints WHERE fingerprint=$1", fingerprint).Scan(&owner); e != nil || owner != str(right["id"]) {
		t.Fatalf("updated donor fingerprint belongs to %s, wanted %s: %v", owner, str(right["id"]), e)
	}
	rawID := hash(str(src["id"]) + ":" + str(donor["url"]) + ":" + str(donor["text"]))
	raw, e := a.store.Get(ctx, "raw_item", rawID)
	if e != nil || str(raw["storyId"]) != str(right["id"]) || str(raw["text"]) != str(donor["text"]) {
		t.Fatal("updated donor raw ownership was not restored")
	}
	rawItems := array(a.request("GET", "/admin/bank/"+str(right["id"])+"/evidence", nil, 200, "")["rawItems"])
	found := false
	for _, item := range rawItems {
		if str(object(item)["id"]) == rawID {
			found = true
		}
	}
	if !found {
		t.Fatal("donor evidence endpoint cannot find updated raw source")
	}
	independent := clone(donor)
	independent["url"] = "https://independent.example/update"
	ingest(independent)
	restored, _ := a.store.Get(ctx, "story", str(right["id"]))
	if len(array(restored["sources"])) != 2 {
		t.Fatal("repeat donor text deduplicated into the wrong story")
	}
	kept, _ := a.store.Get(ctx, "story", str(left["id"]))
	if len(array(kept["sources"])) != 1 {
		t.Fatal("split retained donor-exclusive sources on survivor")
	}
}
