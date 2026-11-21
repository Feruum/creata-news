package news

import (
	"context"
	"testing"
)

func TestDraftJobPayloadSelectsOnlyApprovedTopicLanguageAndDeadlineMatches(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	createOpportunity := func(category, lang, deadline, status string) Doc {
		return a.request("POST", "/admin/opportunities", Doc{"title": localized("Useful grant"), "description": localized("Build a prototype"), "organization": "Creata", "type": "grant", "country": "KZ", "url": "https://example.org/apply", "deadline": deadline, "status": status, "categoryId": category, "languages": []string{lang}}, 200, "")
	}
	eligible := createOpportunity("ai", "en", "2099-01-01T00:00:00Z", "approved")
	createOpportunity("ai", "kk", "2099-01-01T00:00:00Z", "approved")
	createOpportunity("space", "en", "2099-01-01T00:00:00Z", "approved")
	createOpportunity("ai", "en", "2000-01-01T00:00:00Z", "approved")
	createOpportunity("ai", "en", "2099-01-01T00:00:00Z", "pending")
	if e := a.store.Write(ctx, func(tx *Tx) error {
		return tx.Put(ctx, "story", Doc{"id": "english-story", "title": "A source-backed story", "text": "Source text", "url": "https://example.org/source", "sources": []any{Doc{"url": "https://example.org/source"}}, "categoryId": "ai", "locale": "en", "status": "classified"})
	}); e != nil {
		t.Fatal(e)
	}
	article := a.request("POST", "/admin/bank/english-story/draft", nil, 201, "")
	jobs, _ := a.store.List(ctx, "job")
	var queued Doc
	for _, j := range jobs {
		if str(j["kind"]) == "draft" {
			queued = j
		}
	}
	opportunities := array(object(queued["payload"])["opportunities"])
	if len(opportunities) != 1 || str(object(opportunities[0])["id"]) != str(eligible["id"]) {
		t.Fatalf("incorrect opportunity selection: %v", opportunities)
	}
	job := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "fixture", "kinds": []string{"draft"}}, 200, a.worker)["job"])
	result := validArticle()
	result["opportunityIds"] = []string{str(eligible["id"])}
	a.request("POST", "/internal/jobs/"+str(job["id"])+"/complete", Doc{"leaseToken": job["leaseToken"], "result": result}, 200, a.worker)
	article = a.request("GET", "/admin/articles/"+str(article["id"]), nil, 200, "")
	if !has(article["opportunityIds"], str(eligible["id"])) {
		t.Fatal("article lost selected opportunities")
	}
	a.request("POST", "/admin/articles/"+str(article["id"])+"/actions", Doc{"action": "rework", "expectedRevision": article["revision"], "note": "Improve clarity"}, 200, "")
	jobs, _ = a.store.List(ctx, "job")
	for _, j := range jobs {
		if str(j["kind"]) == "rework" {
			if len(array(object(j["payload"])["opportunities"])) != 1 {
				t.Fatal("rework payload omitted opportunities")
			}
		}
	}
}
