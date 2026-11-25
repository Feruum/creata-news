package news

import (
	"context"
	"testing"
)

func TestArticleOpportunityIDsRequireAnActualStringArray(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	for _, bad := range []any{Doc{}, "invented", nil, []any{1}, []any{Doc{"id": "bad"}}} {
		a.request("PATCH", "/admin/articles/"+str(article["id"]), Doc{"expectedRevision": 1, "opportunityIds": bad}, 400, "")
	}
}

func TestOldAIResultCannotOverwriteRejectOrNewerReworkAtSameRevision(t *testing.T) {
	for _, decision := range []string{"reject", "rework", "unpublish"} {
		t.Run(decision, func(t *testing.T) {
			a := setupAPI(t)
			a.login("admin")
			article := a.request("POST", "/admin/articles", validArticle(), 201, "")
			aid := str(article["id"])
			if decision == "unpublish" {
				for _, action := range []string{"approve", "publish"} {
					a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": action, "expectedRevision": 1}, 200, "")
				}
			}
			a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "rework", "expectedRevision": 1, "note": "First request"}, 200, "")
			old := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "older", "kinds": []string{"rework"}}, 200, a.worker)["job"])
			a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": decision, "expectedRevision": 1, "note": "New editorial decision"}, 200, "")
			a.request("POST", "/internal/jobs/"+str(old["id"])+"/complete", Doc{"leaseToken": old["leaseToken"], "result": validArticle()}, 409, a.worker)
			current := a.request("GET", "/admin/articles/"+aid, nil, 200, "")
			if str(current["status"]) != map[string]string{"reject": "rejected", "rework": "rework", "unpublish": "unpublished"}[decision] || integer(current["revision"]) != 1 {
				t.Fatal("Old AI result changed an editorial decision")
			}
			if decision == "rework" {
				newer := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "newer", "kinds": []string{"rework"}}, 200, a.worker)["job"])
				a.request("POST", "/internal/jobs/"+str(newer["id"])+"/complete", Doc{"leaseToken": newer["leaseToken"], "result": validArticle()}, 200, a.worker)
			}
		})
	}
}

func TestBankDraftDoesNotLeakExistingOtherAuthorsArticle(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	if e := a.store.Write(context.Background(), func(tx *Tx) error {
		return tx.Put(context.Background(), "story", Doc{"id": "restricted-story", "articleId": article["id"]})
	}); e != nil {
		t.Fatal(e)
	}
	a.login("author")
	a.request("POST", "/admin/bank/restricted-story/draft", nil, 403, "")
}

func TestClassificationCheckpointRetainsUsageForAccountingRecovery(t *testing.T) {
	a := setupAPI(t)
	ctx := context.Background()
	story := Doc{"id": "paid-classification", "text": "Creata Lab reported a finding.", "url": "https://example.org/source"}
	if e := a.store.Write(ctx, func(tx *Tx) error {
		if e := tx.Put(ctx, "story", story); e != nil {
			return e
		}
		return tx.Put(ctx, "job", newJob("classify", Doc{"story": story}))
	}); e != nil {
		t.Fatal(e)
	}
	j := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "paid", "kinds": []string{"classify"}}, 200, a.worker)["job"])
	output := Doc{"categoryId": "ai", "region": "KZ", "tags": []string{}, "score": 10, "reason": "Supported finding", "facts": []any{}, "entities": []any{}}
	usage := []any{Doc{"reservationId": "pending-usage", "costUsd": 0.01, "inputTokens": 10, "outputTokens": 10}}
	bad := clone(output)
	bad["categoryId"] = "invented-category"
	a.request("POST", "/internal/jobs/"+str(j["id"])+"/checkpoint", Doc{"leaseToken": j["leaseToken"], "phase": "classification", "status": "succeeded", "output": bad, "usage": usage}, 400, a.worker)
	unchanged, _ := a.store.Get(ctx, "job", str(j["id"]))
	if str(object(object(unchanged["checkpoints"])["classification"])["status"]) == "succeeded" {
		t.Fatal("Unknown category poisoned immutable checkpoint")
	}
	a.request("POST", "/internal/jobs/"+str(j["id"])+"/checkpoint", Doc{"leaseToken": j["leaseToken"], "phase": "classification", "status": "succeeded", "output": output, "usage": usage}, 200, a.worker)
	stored, _ := a.store.Get(ctx, "job", str(j["id"]))
	if len(array(object(object(stored["checkpoints"])["classification"])["usage"])) != 1 {
		t.Fatal("paid phase usage lost")
	}
}
