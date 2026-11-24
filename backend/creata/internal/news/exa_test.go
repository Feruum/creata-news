package news

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestExaKeyIsWriteOnlyAndCanBeClearedIndependently(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	const path = "/admin/settings"
	const exaSecret = "exa-fixture-secret"
	const modelSecret = "model-fixture-secret"

	assertPublicState := func(settings Doc, configured bool) {
		t.Helper()
		raw, err := json.Marshal(settings)
		if err != nil {
			t.Fatal(err)
		}
		if object(settings["ai"])["exaKeyConfigured"] != configured {
			t.Fatalf("expected Exa configuration status %t", configured)
		}
		for _, forbidden := range []string{exaSecret, modelSecret, "_exaApiKeyEncrypted", "exaApiKey"} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatal("admin settings exposed a private credential field")
			}
		}
	}
	assertPublicState(a.request("GET", path, nil, 200, ""), false)
	assertPublicState(a.request("PATCH", path, Doc{"ai": Doc{"exaApiKey": exaSecret, "apiKey": modelSecret}}, 200, ""), true)
	assertPublicState(a.request("GET", path, nil, 200, ""), true)
	raw, err := a.store.Get(context.Background(), "settings", "settings")
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext := str(raw["_exaApiKeyEncrypted"]); ciphertext == "" || strings.Contains(ciphertext, exaSecret) {
		t.Fatal("Exa key was not stored encrypted")
	}
	if _, exposed := object(raw["ai"])["exaApiKey"]; exposed {
		t.Fatal("Exa key was stored in general AI settings")
	}

	a.request("GET", "/internal/ai-config", nil, 401, "")
	a.request("GET", "/internal/ai-config", nil, 401, a.bot)
	assertPublicState(a.request("PATCH", path, Doc{"ai": Doc{"model": "changed-model"}}, 200, ""), true)
	a.request("PATCH", path, Doc{"ai": Doc{"exaApiKey": 42}}, 400, "")
	workerConfig := a.request("GET", "/internal/ai-config", nil, 200, a.worker)
	if str(workerConfig["exaApiKey"]) != exaSecret || str(workerConfig["apiKey"]) != modelSecret {
		t.Fatal("the worker did not receive both stored credentials")
	}
	assertPublicState(a.request("PATCH", path, Doc{"ai": Doc{"exaApiKey": ""}}, 200, ""), false)
	workerConfig = a.request("GET", "/internal/ai-config", nil, 200, a.worker)
	if str(workerConfig["exaApiKey"]) != "" || str(workerConfig["apiKey"]) != modelSecret {
		t.Fatal("clearing the Exa key changed the model credential or left Exa enabled")
	}
}

func TestDraftKeepsOnlyCitedExaSourcesWithRetrievedEvidence(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	const primary = "https://example.org/source"
	const supporting = "https://research.example.net/coverage"
	const unused = "https://archive.example.net/unrelated"
	ctx := context.Background()
	if err := a.store.Write(ctx, func(tx *Tx) error {
		return tx.Put(ctx, "story", Doc{
			"id": "exa-story", "title": "An update", "status": "classified", "categoryId": "ai",
			"url": primary, "text": "The editor has an update.", "sources": validArticle()["sources"],
		})
	}); err != nil {
		t.Fatal(err)
	}
	article := a.request("POST", "/admin/bank/exa-story/draft", nil, 201, "")
	job := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "exa-test", "kinds": []string{"draft"}}, 200, a.worker)["job"])
	jobPath := "/internal/jobs/" + str(job["id"])
	brief := Doc{
		"facts": []any{Doc{"claim": "The editor has an update.", "sourceUrls": []string{primary, supporting}}},
		"retrievedEvidence": []any{
			Doc{"url": primary, "title": "Primary", "text": "The editor has an update."},
			Doc{"url": supporting, "title": "Independent coverage", "text": "The editor has an update.", "discoveredBy": "exa"},
			Doc{"url": unused, "title": "Unrelated result", "text": "A different story.", "discoveredBy": "exa"},
		},
		"conflicts": []string{}, "singleSourceClaims": []string{}, "unavailableSources": []string{},
	}
	a.request("POST", jobPath+"/checkpoint", Doc{"leaseToken": job["leaseToken"], "phase": "research", "status": "succeeded", "output": brief}, 200, a.worker)
	draft := validArticle()
	draft["sources"] = []any{Doc{"url": "https://uncited.example.net/unread", "title": "An unverified result"}}
	a.request("POST", jobPath+"/complete", Doc{"leaseToken": job["leaseToken"], "result": draft}, 200, a.worker)

	saved := a.request("GET", "/admin/articles/"+str(article["id"]), nil, 200, "")
	sources := array(saved["sources"])
	if len(sources) != 2 || str(object(sources[0])["url"]) != primary || str(object(sources[1])["url"]) != supporting ||
		str(object(sources[1])["title"]) != "Independent coverage" {
		t.Fatalf("unexpected draft sources: %v", sources)
	}
	a.request("GET", "/public/articles/"+str(saved["slug"]), nil, 404, "")
}
