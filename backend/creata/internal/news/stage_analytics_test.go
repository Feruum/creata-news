package news

import (
	"testing"
	"time"
)

func TestEditorialStageMetricsIncludeDurationsAndFunnel(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	articles := []Doc{{"id": "a", "status": "published", "createdAt": start.Format(time.RFC3339)}, {"id": "b", "status": "review", "createdAt": start.Format(time.RFC3339)}}
	audits := []Doc{{"targetId": "a", "action": "article.transition", "createdAt": start.Add(time.Hour).Format(time.RFC3339), "detail": Doc{"fromStatus": "new", "toStatus": "writing"}}, {"targetId": "a", "action": "article.transition", "createdAt": start.Add(3 * time.Hour).Format(time.RFC3339), "detail": Doc{"fromStatus": "writing", "toStatus": "review"}}, {"targetId": "a", "action": "article.transition", "createdAt": start.Add(4 * time.Hour).Format(time.RFC3339), "detail": Doc{"fromStatus": "review", "toStatus": "approved"}}, {"targetId": "a", "action": "article.transition", "createdAt": start.Add(5 * time.Hour).Format(time.RFC3339), "detail": Doc{"fromStatus": "approved", "toStatus": "published"}}, {"targetId": "b", "action": "article.transition", "createdAt": start.Add(2 * time.Hour).Format(time.RFC3339), "detail": Doc{"fromStatus": "new", "toStatus": "review"}}}
	metrics := editorialStageMetrics(articles, audits, start.Add(8*time.Hour))
	durations := array(metrics["durations"])
	writing := Doc{}
	for _, raw := range durations {
		if str(object(raw)["stage"]) == "writing" {
			writing = object(raw)
		}
	}
	if number(writing["averageSeconds"]) != 7200 || integer(writing["completedVisits"]) != 1 {
		t.Fatalf("bad writing duration: %v", writing)
	}
	funnel := array(metrics["funnel"])
	if len(funnel) != 9 || str(object(funnel[5])["stage"]) != "scheduled" || str(object(funnel[7])["stage"]) != "unpublished" || integer(object(funnel[0])["count"]) != 2 {
		t.Fatal("missing funnel")
	}
	found := false
	for _, raw := range array(metrics["transitions"]) {
		d := object(raw)
		if str(d["from"]) == "new" && str(d["to"]) == "review" && integer(d["count"]) == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("missing transition")
	}
}
