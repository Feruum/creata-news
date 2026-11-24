package news

import (
	"context"
	"reflect"
	"testing"
)

func articlePhotoFixture(filename string) map[string]any {
	return map[string]any{
		"url":        "https://upload.wikimedia.org/wikipedia/commons/a/ab/" + filename,
		"caption":    "Editor screenshot",
		"credit":     "Example author",
		"rights":     "CC BY 4.0",
		"sourceUrl":  "https://commons.wikimedia.org/wiki/File:" + filename,
		"licenseUrl": "https://creativecommons.org/licenses/by/4.0/",
	}
}

func TestHermesPhotosReachDraftWithoutReplacingEditorialSelection(t *testing.T) {
	for _, tc := range []struct {
		name  string
		score int
	}{
		{"automatic", 90},
		{"from the news bank", 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := setupAPI(t)
			a.login("admin")
			ctx := context.Background()
			story := Doc{"id": "story-with-photos", "title": "An editor update", "text": "The editor has an update.", "status": "new", "url": "https://example.org/source", "sources": validArticle()["sources"]}
			if err := a.store.Write(ctx, func(tx *Tx) error {
				if err := tx.Put(ctx, "story", story); err != nil {
					return err
				}
				return tx.Put(ctx, "job", newJob("classify", Doc{"story": story}))
			}); err != nil {
				t.Fatal(err)
			}

			claim := func(kind string) Doc {
				return object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "photos", "kinds": []string{kind}}, 200, a.worker)["job"])
			}
			complete := func(job, result Doc) {
				a.request("POST", "/internal/jobs/"+str(job["id"])+"/complete", Doc{"leaseToken": job["leaseToken"], "result": result}, 200, a.worker)
			}
			photos := []any{articlePhotoFixture("Editor.png"), articlePhotoFixture("Timeline.png")}
			complete(claim("classify"), Doc{"categoryId": "ai", "region": "Global", "tags": []string{}, "score": tc.score, "reason": "Source backed", "photos": photos})
			savedStory, err := a.store.Get(ctx, "story", str(story["id"]))
			if err != nil {
				t.Fatal(err)
			}
			articleID := str(savedStory["articleId"])
			if articleID == "" {
				articleID = str(a.request("POST", "/admin/bank/"+str(story["id"])+"/draft", nil, 201, "")["id"])
			}
			path := "/admin/articles/" + articleID
			article := a.request("GET", path, nil, 200, "")
			if !reflect.DeepEqual(photos, article["photos"]) {
				t.Fatalf("first editorial revision did not include photos with their attribution: %v", article["photos"])
			}
			a.request("GET", "/public/articles/"+str(article["slug"]), nil, 404, "")

			complete(claim("draft"), validArticle())
			article = a.request("GET", path, nil, 200, "")
			if !reflect.DeepEqual(photos, article["photos"]) {
				t.Fatal("the generated text replaced the preselected photos")
			}
			article = a.request("PATCH", path, Doc{"expectedRevision": article["revision"], "photos": []any{}}, 200, "")
			a.request("POST", path+"/actions", Doc{"action": "rework", "expectedRevision": article["revision"], "note": "Shorten the lead"}, 200, "")
			reworked := validArticle()
			reworked["photos"] = photos
			complete(claim("rework"), reworked)
			article = a.request("GET", path, nil, 200, "")
			if items, ok := article["photos"].([]any); !ok || len(items) != 0 {
				t.Fatalf("a rework restored photos that the editor removed: %v", article["photos"])
			}
		})
	}
}

func TestArticlePhotosRequireDistinctWikimediaFilesWithMatchingLicenses(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	path := "/admin/articles/" + str(article["id"])
	valid := articlePhotoFixture("Editor.png")
	for _, field := range []string{"url", "sourceUrl", "licenseUrl", "credit", "rights"} {
		invalid := articlePhotoFixture("Timeline.png")
		invalid[field] = "https://example.org/wrong"
		if field == "credit" {
			invalid[field] = ""
		}
		a.request("PATCH", path, Doc{"expectedRevision": 1, "photos": []any{invalid, valid}}, 400, "")
	}
	a.request("PATCH", path, Doc{"expectedRevision": 1, "photos": []any{valid, valid}}, 400, "")
	duplicate := articlePhotoFixture("Timeline.png")
	duplicate["url"] = valid["url"]
	a.request("PATCH", path, Doc{"expectedRevision": 1, "photos": []any{valid, duplicate}}, 400, "")
	saved := a.request("PATCH", path, Doc{"expectedRevision": 1, "photos": []any{valid}}, 200, "")
	a.request("POST", path+"/actions", Doc{"action": "approve", "expectedRevision": saved["revision"]}, 200, "")
	a.request("POST", path+"/actions", Doc{"action": "publish", "expectedRevision": saved["revision"]}, 200, "")
	public := a.request("GET", "/public/articles/"+str(saved["slug"]), nil, 200, "")
	if !reflect.DeepEqual([]any{valid}, public["photos"]) {
		t.Fatalf("public article lost the selected photo or attribution: %v", public["photos"])
	}
}
