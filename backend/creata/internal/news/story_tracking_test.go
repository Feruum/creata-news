package news

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
)

const followedStoriesPath = "/account/followed-stories"

func createTrackedStory(a *apiTest, fields Doc) Doc {
	input := validArticle()
	for key, value := range fields {
		input[key] = value
	}
	return a.request("POST", "/admin/articles", input, 201, "")
}

func publishTrackedStory(a *apiTest, path string, revision int) {
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", path+"/actions", Doc{"action": action, "expectedRevision": revision}, 200, "")
	}
}

func assertUnreadStoryUpdates(a *apiTest, articleID string, want ...string) {
	a.t.Helper()
	feed := a.request("GET", followedStoriesPath, nil, 200, "")
	unread, ok := feed["unreadUpdateIds"].(map[string]any)
	if !ok {
		a.t.Fatalf("unreadUpdateIds must be an object, got %v", feed["unreadUpdateIds"])
	}
	ids, ok := unread[articleID].([]any)
	if !ok {
		a.t.Fatalf("unreadUpdateIds[%q] must be an array, got %v", articleID, unread[articleID])
	}
	got := make([]string, 0, len(ids))
	for _, value := range ids {
		got = append(got, str(value))
	}
	if !slices.Equal(got, want) {
		a.t.Fatalf("unread updates for %q = %q, want %q", articleID, got, want)
	}
}

func TestAPIFollowedStoryReadTracksPublishedUpdateIDs(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	initial := storyUpdate("first", "2026-03-01T09:00:00Z")
	second := storyUpdate("second", "2026-03-02T09:00:00Z")
	third := storyUpdate("third", "2026-03-03T09:00:00Z")
	article := createTrackedStory(a, Doc{"storyUpdates": []any{initial}})
	articleID := str(article["id"])
	articlePath, followPath := "/admin/articles/"+articleID, followedStoriesPath+"/"+articleID
	publishTrackedStory(a, articlePath, 1)
	a.login("reader")
	a.request("POST", followPath, nil, 200, "")
	assertUnreadStoryUpdates(a, articleID)

	a.login("admin")
	initial["title"] = localized("Updated headline for the same timeline entry")
	a.request("PATCH", articlePath, Doc{"expectedRevision": 1, "storyUpdates": []any{initial, second, third}}, 200, "")
	a.login("reader")
	assertUnreadStoryUpdates(a, articleID)
	a.request("POST", followPath+"/read", Doc{"updateIds": []string{"second"}}, 200, "")

	a.login("admin")
	publishTrackedStory(a, articlePath, 2)
	a.login("reader")
	a.request("POST", followPath, nil, 200, "")
	assertUnreadStoryUpdates(a, articleID, "third", "second")
	a.request("POST", followPath+"/read", Doc{"updateIds": []string{"second", "second"}}, 200, "")
	assertUnreadStoryUpdates(a, articleID, "third")
	a.request("POST", followPath+"/read", Doc{"updateIds": []string{"second"}}, 200, "")
	assertUnreadStoryUpdates(a, articleID, "third")

	a.login("admin")
	a.request("POST", articlePath+"/actions", Doc{"action": "unpublish", "expectedRevision": 2}, 200, "")
	a.login("reader")
	a.request("POST", followPath+"/read", Doc{"updateIds": []string{"third"}}, 404, "")
	feed := a.request("GET", followedStoriesPath, nil, 200, "")
	if len(array(feed["items"])) != 0 || len(object(feed["unreadUpdateIds"])) != 0 {
		t.Fatalf("withdrawn story or unread IDs exposed: %v", feed)
	}

	a.login("admin")
	publishTrackedStory(a, articlePath, 2)
	a.login("reader")
	assertUnreadStoryUpdates(a, articleID, "third")
}

func TestAPILegacyStoryFollowReadMergesConcurrentRequests(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := createTrackedStory(a, Doc{"storyUpdates": []any{
		storyUpdate("first", "2026-03-01T09:00:00Z"),
		storyUpdate("second", "2026-03-02T09:00:00Z"),
	}})
	articleID := str(article["id"])
	publishTrackedStory(a, "/admin/articles/"+articleID, 1)
	a.login("reader")
	ctx := context.Background()
	if err := a.store.Write(ctx, func(tx *Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO creata_records(kind, id, data)
			VALUES('story_follow', $1, jsonb_build_object('userId', 'dev-reader', 'articleId', $2::text))`,
			hash("dev-reader\x00"+articleID), articleID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	assertUnreadStoryUpdates(a, articleID, "second", "first")

	path := followedStoriesPath + "/" + articleID + "/read"
	a.login("author")
	a.request("POST", path, Doc{"updateIds": []string{"first"}}, 404, "")
	a.login("reader")
	assertUnreadStoryUpdates(a, articleID, "second", "first")
	results := make(chan error, 2)
	for _, id := range []string{"first", "second"} {
		payload, _ := json.Marshal(Doc{"updateIds": []string{id}})
		request, err := http.NewRequest(http.MethodPost, a.server.URL+base+path, bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-CSRF-Token", a.csrf)
		go func() {
			response, err := a.client.Do(request)
			if err == nil {
				response.Body.Close()
				if response.StatusCode != 200 {
					t.Errorf("mark-read returned HTTP %d", response.StatusCode)
				}
			}
			results <- err
		}()
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	assertUnreadStoryUpdates(a, articleID)
}

func TestAPIFollowedStoryReadValidatesRequest(t *testing.T) {
	a := setupAPI(t)
	path := followedStoriesPath + "/unknown/read"
	a.request("POST", path, Doc{"updateIds": []string{}}, 401, "")
	a.login("reader")
	for _, updateIDs := range []any{nil, "first", []any{1}, []any{"../first"}, []any{strings.Repeat("a", 81)}, slices.Repeat([]string{"first"}, 41)} {
		a.request("POST", path, Doc{"updateIds": updateIDs}, 400, "")
	}
	a.request("POST", path, Doc{}, 400, "")
	a.request("POST", path, Doc{"updateIds": []string{}, "markAll": true}, 400, "")
	csrf := a.csrf
	a.csrf = ""
	a.request("POST", path, Doc{"updateIds": []string{}}, 403, "")
	a.csrf = csrf
	a.request("POST", path, Doc{"updateIds": []string{}}, 404, "")
}

func TestAPIStoryFollowUpSchedulePreservesArticleRevisions(t *testing.T) {
	a := setupAPI(t)
	const queue = "/admin/story-follow-ups"
	a.request("GET", queue, nil, 401, "")
	a.login("admin")
	first := createTrackedStory(a, Doc{"title": localized("First story"), "storyUpdates": []any{storyUpdate("announcement", "2026-03-01T09:00:00Z")}})
	second := createTrackedStory(a, Doc{"title": localized("Second story"), "storyUpdates": []any{storyUpdate("announcement", "2026-03-02T09:00:00Z")}})
	firstID, secondID := str(first["id"]), str(second["id"])
	firstArticle, secondArticle := "/admin/articles/"+firstID, "/admin/articles/"+secondID
	firstFollowUp, secondFollowUp := firstArticle+"/follow-up", secondArticle+"/follow-up"
	if state := a.request("GET", firstFollowUp, nil, 200, ""); state["item"] != nil || state["eligible"] != true {
		t.Fatalf("unexpected unscheduled story state: %v", state)
	}
	for _, date := range []string{"2026-02-29", "2026-03-1", "2026-03-01T00:00:00Z", "0000-03-01"} {
		a.request("PUT", firstFollowUp, Doc{"assigneeId": "dev-admin", "dueDate": date}, 400, "")
	}
	firstItem := object(a.request("PUT", firstFollowUp, Doc{"assigneeId": "dev-admin", "dueDate": "2028-02-29"}, 200, "")["item"])
	if str(firstItem["articleId"]) != firstID || localize(firstItem["title"], "en") != "First story" || str(firstItem["assigneeName"]) != "Development admin" || str(firstItem["assigneeId"]) != "dev-admin" || str(firstItem["dueDate"]) != "2028-02-29" || parseTime(firstItem["updatedAt"]).IsZero() {
		t.Fatalf("unexpected follow-up: %v", firstItem)
	}
	a.request("PUT", secondFollowUp, Doc{"assigneeId": "dev-admin", "dueDate": "2026-03-28"}, 200, "")
	list := a.request("GET", queue, nil, 200, "")
	items := array(list["items"])
	if integer(list["total"]) != 2 || len(items) != 2 || str(object(items[0])["articleId"]) != secondID || str(object(items[1])["articleId"]) != firstID {
		t.Fatalf("queue is not ordered by due date: %v", list)
	}
	current := a.request("GET", firstArticle, nil, 200, "")
	if current["revision"] != first["revision"] || current["updatedAt"] != first["updatedAt"] || current["publishedRevision"] != nil {
		t.Fatalf("scheduling changed article state: %v", current)
	}
	publishTrackedStory(a, firstArticle, 1)
	a.request("PATCH", firstArticle, Doc{"expectedRevision": 1, "storyUpdates": []any{}}, 200, "")
	if state := a.request("GET", firstFollowUp, nil, 200, ""); state["eligible"] != true {
		t.Fatalf("published chronology should allow rescheduling: %v", state)
	}
	a.request("PUT", firstFollowUp, Doc{"assigneeId": "dev-admin", "dueDate": "2026-03-27"}, 200, "")
	if public := a.request("GET", "/public/articles/"+str(first["slug"]), nil, 200, ""); public["followUp"] != nil || public["dueDate"] != nil || public["assigneeId"] != nil {
		t.Fatalf("public article exposed editorial schedule: %v", public)
	}
	a.request("POST", firstArticle+"/actions", Doc{"action": "unpublish", "expectedRevision": 2}, 200, "")
	a.request("PUT", firstFollowUp, Doc{"assigneeId": "dev-admin", "dueDate": "2026-03-26"}, 400, "")
	if state := a.request("GET", firstFollowUp, nil, 200, ""); state["eligible"] != false || str(object(state["item"])["dueDate"]) != "2026-03-27" {
		t.Fatalf("pending follow-up state after withdrawal: %v", state)
	}
	for range 2 {
		a.request("DELETE", firstFollowUp, nil, 200, "")
	}
	if item := a.request("GET", firstFollowUp, nil, 200, "")["item"]; item != nil {
		t.Fatalf("completed story remained scheduled: %v", item)
	}
	counts := map[string]int{}
	for _, entry := range array(a.request("GET", "/admin/audit", nil, 200, "")["items"]) {
		audit := object(entry)
		if str(audit["targetId"]) == firstID {
			counts[str(audit["action"])]++
		}
	}
	if counts["story_follow_up.set"] != 2 || counts["story_follow_up.complete"] != 1 {
		t.Fatalf("unexpected audit counts for follow-up actions: %v", counts)
	}
}

func TestAPIStoryFollowUpAuthorizationUsesArticleScope(t *testing.T) {
	a := setupAPI(t)
	a.login("editor")
	a.login("author")
	a.request("GET", "/admin/story-follow-ups", nil, 403, "")
	a.login("admin")
	a.request("PATCH", "/admin/users/dev-editor", Doc{"verticals": []string{"ai"}}, 200, "")
	followupPath := func(category, layer string) string {
		article := createTrackedStory(a, Doc{"categoryId": category, "layer": layer, "storyUpdates": []any{storyUpdate("one", "2026-03-03T09:00:00Z")}})
		return "/admin/articles/" + str(article["id"]) + "/follow-up"
	}
	allowed, other, eco := followupPath("ai", "world"), followupPath("science", "world"), followupPath("ai", "eco")
	for _, path := range []string{allowed, other, eco} {
		a.request("PUT", path, Doc{"assigneeId": "dev-admin", "dueDate": "2027-02-15"}, 200, "")
	}
	for _, recipient := range []string{"dev-author", "nonexistent"} {
		a.request("PUT", allowed, Doc{"assigneeId": recipient, "dueDate": "2027-02-15"}, 400, "")
	}
	for _, path := range []string{other, eco} {
		a.request("PUT", path, Doc{"assigneeId": "dev-editor", "dueDate": "2027-02-15"}, 400, "")
	}
	a.login("editor")
	list := a.request("GET", "/admin/story-follow-ups", nil, 200, "")
	items := array(list["items"])
	if len(items) != 1 || integer(list["total"]) != 1 || allowed != "/admin/articles/"+str(object(items[0])["articleId"])+"/follow-up" {
		t.Fatalf("queue visibility exceeded the editor's article scope: %v", list)
	}
	for _, path := range []string{other, eco} {
		a.request("GET", path, nil, 403, "")
		a.request("PUT", path, Doc{"assigneeId": "dev-admin", "dueDate": "2027-02-15"}, 403, "")
		a.request("DELETE", path, nil, 403, "")
	}
	a.request("PUT", allowed, Doc{"assigneeId": "dev-editor", "dueDate": "2027-02-16"}, 200, "")
}

func TestAPIStoryFollowUpAllowsCrossOriginPut(t *testing.T) {
	a := setupAPI(t)
	origin := "http://localhost:6868"
	a.app.Config.Origins = []string{origin}
	request, err := http.NewRequest(http.MethodOptions, a.server.URL+base+"/admin/articles/article/follow-up", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", origin)
	request.Header.Set("Access-Control-Request-Method", http.MethodPut)
	response, err := a.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if !slices.Contains(strings.Split(response.Header.Get("Access-Control-Allow-Methods"), ","), http.MethodPut) {
		t.Fatal("preflight does not allow PUT")
	}
}
