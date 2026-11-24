package news

import "testing"

func TestFounderLaunchSubmissionModeration(t *testing.T) {
	a := setupAPI(t)
	const account = "/account/launch-submissions"
	const admin = "/admin/launch-submissions"
	a.request("GET", account, nil, 401, "")
	a.login("reader")
	draft := a.request("POST", account, Doc{"name": "Creata Studio", "status": "published", "ownerId": "dev-author"}, 201, "")
	submissionID := str(draft["id"])
	path := account + "/" + submissionID
	if str(draft["status"]) != "draft" || len(array(a.request("GET", account, nil, 200, "")["items"])) != 1 {
		t.Fatal("the founder must see their own private draft")
	}
	a.request("POST", path+"/submit", Doc{"expectedRevision": 1}, 400, "")
	a.request("GET", admin, nil, 403, "")

	a.login("author")
	if len(array(a.request("GET", account, nil, 200, "")["items"])) != 0 {
		t.Fatal("another account saw the founder's draft")
	}
	a.request("PATCH", path, Doc{"expectedRevision": 1, "name": "Other account"}, 404, "")
	a.request("GET", admin, nil, 403, "")

	a.login("reader")
	fields := validLaunch()
	fields["expectedRevision"] = 1
	csrf := a.csrf
	a.csrf = ""
	a.request("PATCH", path, fields, 403, "")
	a.csrf = csrf
	a.request("PATCH", path, fields, 200, "")
	a.request("POST", path+"/submit", Doc{"expectedRevision": 1}, 409, "")
	submitted := a.request("POST", path+"/submit", Doc{"expectedRevision": 2}, 200, "")
	if str(submitted["status"]) != "in_review" {
		t.Fatal(submitted)
	}
	a.request("PATCH", path, Doc{"expectedRevision": 3, "name": "Edited in review"}, 409, "")
	if len(array(a.request("GET", "/public/launches", nil, 200, "")["items"])) != 0 {
		t.Fatal("unapproved content leaked to the public catalog")
	}

	a.login("editor")
	review := admin + "/" + submissionID + "/review"
	if len(array(a.request("GET", admin, nil, 200, "")["items"])) != 1 {
		t.Fatal("the submitted launch is missing from the editorial queue")
	}
	a.request("POST", review, Doc{"action": "request_changes", "expectedRevision": 3, "note": "  "}, 400, "")
	a.request("POST", review, Doc{"action": "request_changes", "expectedRevision": 3, "note": "Add more detail about the team."}, 200, "")

	a.login("reader")
	items := array(a.request("GET", account, nil, 200, "")["items"])
	returned := object(items[0])
	if str(returned["status"]) != "changes_requested" || str(returned["reviewNote"]) != "Add more detail about the team." {
		t.Fatal(returned)
	}
	a.request("PATCH", path, Doc{"expectedRevision": 4, "team": localized("Two engineers in Almaty")}, 200, "")
	a.request("POST", path+"/submit", Doc{"expectedRevision": 5}, 200, "")

	a.login("editor")
	a.request("POST", review, Doc{"action": "publish", "expectedRevision": 3}, 409, "")
	published := a.request("POST", review, Doc{"action": "publish", "expectedRevision": 6}, 200, "")
	if str(published["status"]) != "published" || str(published["launchSlug"]) == "" {
		t.Fatal(published)
	}
	launch := a.request("GET", "/public/launches/"+str(published["launchSlug"]), nil, 200, "")
	if localize(launch["team"], "en") != "Two engineers in Almaty" || launch["ownerId"] != nil || launch["reviewNote"] != nil {
		t.Fatal("the public launch must contain the reviewed content and no private submission metadata")
	}
	a.request("POST", review, Doc{"action": "publish", "expectedRevision": 6}, 409, "")
	if len(array(a.request("GET", "/public/launches", nil, 200, "")["items"])) != 1 {
		t.Fatal("repeat approval must not create duplicates")
	}
	a.login("reader")
	a.request("POST", "/account/launches/"+str(launch["id"])+"/vote", Doc{"voted": true}, 403, "")

	a.login("editor")
	a.request("PATCH", "/admin/launches/"+str(launch["id"]), Doc{"expectedRevision": 1, "status": "archived"}, 200, "")
	a.login("reader")
	archived := object(array(a.request("GET", account, nil, 200, "")["items"])[0])
	if str(archived["status"]) != "unpublished" || archived["launchSlug"] != nil {
		t.Fatal("the founder profile still links to an archived launch")
	}
}
