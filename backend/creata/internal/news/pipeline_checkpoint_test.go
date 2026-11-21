package news

import (
	"context"
	"testing"
)

func TestPipelineCheckpointSurvivesRetryAndRejectsStaleLease(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	a.request("POST", "/admin/articles/"+str(article["id"])+"/actions", Doc{"action": "rework", "expectedRevision": 1, "note": "Check facts"}, 200, "")
	j := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "first", "kinds": []string{"rework"}}, 200, a.worker)["job"])
	path := "/internal/jobs/" + str(j["id"])
	brief := Doc{"facts": []any{Doc{"claim": "Source reports a finding", "sourceUrls": []string{"https://example.org/source"}}}, "conflicts": []any{}, "singleSourceClaims": []string{"Source reports a finding"}, "retrievedEvidence": []any{Doc{"url": "https://example.org/source", "text": "Source reports a finding"}}, "unavailableSources": []any{}}
	a.request("POST", path+"/checkpoint", Doc{"leaseToken": "wrong", "phase": "research", "status": "succeeded", "output": brief}, 409, a.worker)
	a.request("POST", path+"/checkpoint", Doc{"leaseToken": j["leaseToken"], "phase": "research", "status": "succeeded", "output": brief}, 200, a.worker)
	a.request("POST", path+"/checkpoint", Doc{"leaseToken": j["leaseToken"], "phase": "draft", "status": "failed", "error": "Interrupted"}, 200, a.worker)
	a.request("POST", path+"/fail", Doc{"leaseToken": j["leaseToken"], "error": "Interrupted"}, 200, a.worker)
	a.request("POST", "/admin/jobs/"+str(j["id"])+"/retry", nil, 200, "")
	resumed := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "second", "kinds": []string{"rework"}}, 200, a.worker)["job"])
	if str(object(object(resumed["checkpoints"])["research"])["status"]) != "succeeded" {
		t.Fatal("research checkpoint lost")
	}
	a.request("POST", path+"/checkpoint", Doc{"leaseToken": j["leaseToken"], "phase": "review", "status": "failed"}, 409, a.worker)
	if err := a.store.Write(context.Background(), func(tx *Tx) error {
		resumed["leaseUntil"] = "2000-01-01T00:00:00Z"
		return tx.Put(context.Background(), "job", resumed)
	}); err != nil {
		t.Fatal(err)
	}
	a.request("POST", path+"/heartbeat", Doc{"leaseToken": resumed["leaseToken"]}, 409, a.worker)
	a.request("POST", path+"/checkpoint", Doc{"leaseToken": resumed["leaseToken"], "phase": "draft", "status": "succeeded", "output": validArticle()}, 409, a.worker)
}

func TestPipelineRejectsInventedOpportunityIDs(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	a.request("POST", "/admin/articles/"+str(article["id"])+"/actions", Doc{"action": "rework", "expectedRevision": 1, "note": "Check"}, 200, "")
	j := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "worker", "kinds": []string{"rework"}}, 200, a.worker)["job"])
	result := validArticle()
	result["opportunityIds"] = []string{"invented"}
	a.request("POST", "/internal/jobs/"+str(j["id"])+"/complete", Doc{"leaseToken": j["leaseToken"], "result": result}, 400, a.worker)
}
