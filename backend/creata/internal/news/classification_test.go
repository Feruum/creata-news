package news

import (
	"context"
	"testing"
)

func TestClassificationRejectsMalformedOrUnretrievedFactsAndEntities(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	url := "https://example.org/source"
	story := Doc{"id": "classification-source", "title": "Creata Lab launched Orbit One", "text": "Creata Lab launched Orbit One in 2026. The prototype reached 120 users.", "url": url, "locale": "en", "status": "new", "sources": []any{Doc{"url": url}, Doc{"url": "https://example.org/no-text"}}}
	if e := a.store.Write(ctx, func(tx *Tx) error {
		if e := tx.Put(ctx, "story", story); e != nil {
			return e
		}
		return tx.Put(ctx, "job", newJob("classify", Doc{"story": story}))
	}); e != nil {
		t.Fatal(e)
	}
	job := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "classification", "kinds": []string{"classify"}}, 200, a.worker)["job"])
	valid := func() Doc {
		return Doc{"categoryId": "ai", "region": "KZ", "tags": []string{"research"}, "score": 10, "reason": "A source-backed finding", "facts": []any{Doc{"claim": "The prototype reached 120 users.", "sourceUrls": []string{url}, "uncertainty": "Single source"}}, "entities": []any{Doc{"name": "Creata Lab", "type": "organization", "sourceUrls": []string{url}}}}
	}
	invalid := []Doc{{"facts": "not-an-array"}, {"entities": []string{"Creata Lab"}}, {"facts": []any{Doc{"claim": "The prototype reached 999 users.", "sourceUrls": []string{url}, "uncertainty": ""}}}, {"facts": []any{Doc{"claim": "The prototype reached 120 users.", "sourceUrls": []string{"https://example.org/no-text"}, "uncertainty": ""}}}, {"facts": []any{Doc{"claim": "The prototype reached 120 users.", "sourceUrls": []string{url}}}}, {"entities": []any{Doc{"name": "Invented Labs", "type": "organization", "sourceUrls": []string{url}}}}, {"entities": []any{Doc{"name": "Creat", "type": "organization", "sourceUrls": []string{url}}}}, {"entities": []any{Doc{"name": "Creata Lab", "type": "unknown", "sourceUrls": []string{url}}}}, {"tags": []any{Doc{"invalid": true}}}, {"score": true}}
	for _, change := range invalid {
		result := valid()
		for k, v := range change {
			result[k] = v
		}
		a.request("POST", "/internal/jobs/"+str(job["id"])+"/complete", Doc{"leaseToken": job["leaseToken"], "result": result}, 400, a.worker)
	}
	kept, _ := a.store.Get(ctx, "story", str(story["id"]))
	if str(kept["status"]) != "new" {
		t.Fatal("invalid classification changed story")
	}
	a.request("POST", "/internal/jobs/"+str(job["id"])+"/complete", Doc{"leaseToken": job["leaseToken"], "result": valid()}, 200, a.worker)
	saved, _ := a.store.Get(ctx, "story", str(story["id"]))
	if len(array(saved["facts"])) != 1 || len(array(saved["entities"])) != 1 {
		t.Fatal("valid extracted provenance was not persisted")
	}
}

func TestLegacyClassificationNormalizesOmittedListsAndKeepsManualDraft(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	story := Doc{"id": "manual-draft-story", "title": "Source title", "text": "Retained source text", "url": "https://example.org/source", "status": "in_progress", "articleId": article["id"], "categoryId": "ai"}
	if e := a.store.Write(ctx, func(tx *Tx) error {
		if e := tx.Put(ctx, "story", story); e != nil {
			return e
		}
		return tx.Put(ctx, "job", newJob("classify", Doc{"story": story}))
	}); e != nil {
		t.Fatal(e)
	}
	job := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "classification", "kinds": []string{"classify"}}, 200, a.worker)["job"])
	a.request("POST", "/internal/jobs/"+str(job["id"])+"/complete", Doc{"leaseToken": job["leaseToken"], "result": Doc{"categoryId": "science", "region": "KZ", "tags": []string{}, "score": 90, "reason": "Legacy fixture"}}, 200, a.worker)
	saved, _ := a.store.Get(ctx, "story", str(story["id"]))
	if str(saved["status"]) != "in_progress" || saved["facts"] == nil || saved["entities"] == nil {
		t.Fatal("legacy normalization lost manual editorial state")
	}
	unchanged := a.request("GET", "/admin/articles/"+str(article["id"]), nil, 200, "")
	if integer(unchanged["revision"]) != 1 || str(unchanged["categoryId"]) != "ai" || localize(unchanged["title"], "en") != "A tested story" {
		t.Fatal("classification changed the manual article")
	}
	jobs, _ := a.store.List(ctx, "job")
	if len(jobs) != 1 {
		t.Fatal("classification generated a duplicate draft")
	}
}
