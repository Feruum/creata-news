package news

import (
	"context"
	"testing"
)

func testGraphic() Doc {
	return Doc{"type": "bar", "unit": "USD million", "period": "2024–2025", "sourceUrl": "https://example.org/report", "sourceName": "Annual report", "methodology": localized("Reported amounts, same currency and period."), "verified": false, "labels": []any{localized("2024"), localized("2025")}, "series": []any{Doc{"name": localized("Revenue"), "values": []any{100.0, 120.0}}}}
}

func TestGraphicRejectsMisleadingOrInvalidDatasets(t *testing.T) {
	for _, mutate := range []func(Doc){
		func(g Doc) { object(array(g["series"])[0])["values"] = []any{1.0} },
		func(g Doc) { object(array(g["series"])[0])["values"] = []any{"100", 120.0} },
		func(g Doc) { g["sourceUrl"] = "javascript:alert(1)" },
		func(g Doc) { g["type"] = "donut"; object(array(g["series"])[0])["values"] = []any{-1.0, 2.0} },
		func(g Doc) { g["type"] = "donut"; object(array(g["series"])[0])["values"] = []any{0.0, 0.0} },
		func(g Doc) { g["labels"] = []any{Doc{"ru": "Only one language"}, localized("2025")} },
	} {
		g := testGraphic()
		mutate(g)
		if validateGraphic(g, false) == nil {
			t.Fatal("Invalid financial dataset accepted", g)
		}
	}
	if e := validateGraphic(testGraphic(), false); e != nil {
		t.Fatal(e)
	}
}

func TestGraphicPublicationRequiresVerificationAndUsesApprovedSnapshot(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	input := validArticle()
	input["format"] = "graphic"
	input["graphic"] = testGraphic()
	article := a.request("POST", "/admin/articles", input, 201, "")
	aid := str(article["id"])
	a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "approve", "expectedRevision": 1}, 400, "")
	g := testGraphic()
	g["verified"] = true
	a.request("PATCH", "/admin/articles/"+aid, Doc{"expectedRevision": 1, "graphic": g, "sources": []any{Doc{"url": g["sourceUrl"], "title": g["sourceName"]}}}, 200, "")
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": action, "expectedRevision": 2}, 200, "")
	}
	items := array(a.request("GET", "/public/articles?format=graphic", nil, 200, "")["items"])
	if len(items) != 1 || len(object(object(items[0])["graphic"])) == 0 {
		t.Fatal("Published graphic is missing")
	}
	g["verified"] = false
	object(array(g["series"])[0])["values"] = []any{999.0, 999.0}
	a.request("PATCH", "/admin/articles/"+aid, Doc{"expectedRevision": 2, "graphic": g}, 200, "")
	pub := a.request("GET", "/public/articles/"+str(article["slug"]), nil, 200, "")
	if number(array(object(array(object(pub["graphic"])["series"])[0])["values"])[0]) != 100 {
		t.Fatal("Unapproved data leaked")
	}
	a.request("PATCH", "/admin/articles/"+aid, Doc{"expectedRevision": 2, "graphic": g}, 409, "")
}

func TestHermesGraphicCannotPublishOrCarryVerification(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	input := validArticle()
	input["format"] = "graphic"
	article := a.request("POST", "/admin/articles", input, 201, "")
	aid := str(article["id"])
	a.request("POST", "/admin/articles/"+aid+"/graphic-draft", Doc{"expectedRevision": 1}, 202, "")
	job := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "graphic-test", "kinds": []any{"draft"}}, 200, a.worker)["job"])
	result := validArticle()
	result["graphic"] = testGraphic()
	object(result["graphic"])["verified"] = true
	a.request("POST", "/internal/jobs/"+str(job["id"])+"/complete", Doc{"leaseToken": job["leaseToken"], "result": result}, 200, a.worker)
	updated := a.request("GET", "/admin/articles/"+aid, nil, 200, "")
	if boolean(object(updated["graphic"])["verified"]) || str(updated["status"]) != "review" {
		t.Fatal("Hermes bypassed data review")
	}
	if len(array(a.request("GET", "/public/articles?format=graphic", nil, 200, "")["items"])) != 0 {
		t.Fatal("AI graphic leaked")
	}
}

func TestGraphicStoryAndDraftQueueAreIdempotent(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	story := Doc{"id": id(), "title": "Annual data report", "categoryId": "ai", "sources": []any{Doc{"url": "https://example.org/report", "title": "Report"}}}
	if e := a.store.Write(context.Background(), func(tx *Tx) error { return tx.Put(context.Background(), "story", story) }); e != nil {
		t.Fatal(e)
	}
	first := a.request("POST", "/admin/bank/"+str(story["id"])+"/graphic", nil, 201, "")
	again := a.request("POST", "/admin/bank/"+str(story["id"])+"/graphic", nil, 201, "")
	if first["id"] != again["id"] || str(first["format"]) != "graphic" {
		t.Fatal("Story retry duplicated its graphic")
	}
	jobs := array(a.request("GET", "/admin/jobs", nil, 200, "")["items"])
	queued := a.request("POST", "/admin/articles/"+str(first["id"])+"/graphic-draft", Doc{"expectedRevision": 1}, 202, "")
	if len(jobs) != 1 || object(jobs[0])["id"] != queued["id"] {
		t.Fatal("Retry duplicated graphic AI job")
	}
}
