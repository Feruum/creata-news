package news

import "testing"

func TestApprovedOpportunityRequiresCompleteTranslationsAndPhotoAttribution(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	draft := a.request("POST", "/admin/opportunities", Doc{"title": Doc{"ru": "Draft"}}, 200, "")
	a.request("PATCH", "/admin/opportunities/"+str(draft["id"]), Doc{"status": "approved"}, 400, "")
	complete := Doc{"title": localized("Opportunity"), "description": localized("Details"), "organization": "Creata", "type": "grant", "country": "KZ", "url": "https://example.org/apply", "deadline": "2099-01-01T00:00:00Z", "status": "approved", "categoryId": "ai", "photo": "/photo.jpg"}
	a.request("POST", "/admin/opportunities", complete, 400, "")
	complete["photoRights"] = Doc{"credit": "Photographer", "rights": "Permission granted"}
	created := a.request("POST", "/admin/opportunities", complete, 200, "")
	a.request("PATCH", "/admin/opportunities/"+str(created["id"]), Doc{"title": Doc{"ru": "Only one language"}}, 400, "")
}

func TestArticleRejectsMissingAuthorAndCategoryReferences(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	input := validArticle()
	input["authorId"] = "missing-author"
	a.request("POST", "/admin/articles", input, 400, "")
	input = validArticle()
	input["categoryId"] = "missing-category"
	a.request("POST", "/admin/articles", input, 400, "")
	created := a.request("POST", "/admin/articles", validArticle(), 201, "")
	a.request("PATCH", "/admin/articles/"+str(created["id"]), Doc{"expectedRevision": 1, "authorId": "missing-author"}, 400, "")
}
