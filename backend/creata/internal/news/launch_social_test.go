package news

import "testing"

func TestLaunchVotesAndDiscussion(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	input := validLaunch()
	input["status"] = "published"
	launch := a.request("POST", "/admin/launches", input, 200, "")
	path := "/account/launches/" + str(launch["id"])
	a.login("reader")
	for i := 0; i < 3; i++ {
		vote := a.request("POST", path+"/vote", Doc{"voted": true}, 200, "")
		if integer(vote["votes"]) != 1 {
			t.Fatal("repeat created duplicate vote")
		}
	}
	if len(array(a.request("GET", "/account/launch-votes", nil, 200, "")["items"])) != 1 {
		t.Fatal("saved vote state missing")
	}
	comment := Doc{"body": "Useful product!", "clientId": "comment-1"}
	created := a.request("POST", path+"/comments", comment, 200, "")
	for i := 0; i < 2; i++ {
		a.request("POST", path+"/comments", comment, 200, "")
	}
	public := a.request("GET", "/public/launches/creata-studio", nil, 200, "")
	if integer(public["votes"]) != 1 || integer(public["commentsCount"]) != 1 {
		t.Fatal("incorrect public counters", public)
	}
	comments := a.request("GET", "/public/launches/creata-studio/comments", nil, 200, "")
	if integer(comments["total"]) != 1 || !boolean(object(array(comments["items"])[0])["canDelete"]) {
		t.Fatal("incorrect comment owner state")
	}
	a.request("POST", path+"/comments", Doc{"body": " ", "clientId": "invalid"}, 400, "")
	a.request("GET", "/public/launches/creata-studio/comments?offset=-1", nil, 400, "")
	a.login("author")
	a.request("DELETE", path+"/comments/"+str(created["id"]), nil, 403, "")
	a.login("reader")
	a.request("DELETE", path+"/comments/"+str(created["id"]), nil, 200, "")
	if integer(a.request("POST", path+"/vote", Doc{"voted": false}, 200, "")["votes"]) != 0 {
		t.Fatal("vote could not be removed")
	}
	a.login("admin")
	a.request("PATCH", "/admin/launches/"+str(launch["id"]), Doc{"expectedRevision": 1, "status": "archived"}, 200, "")
	a.request("GET", "/public/launches/creata-studio/comments", nil, 404, "")
	a.request("POST", path+"/vote", Doc{"voted": true}, 404, "")
	a.request("POST", path+"/comments", Doc{"body": "Hidden project", "clientId": "comment-2"}, 404, "")
	a.request("POST", "/auth/logout", nil, 200, "")
	a.csrf = ""
	a.request("POST", path+"/vote", Doc{"voted": true}, 401, "")
}
