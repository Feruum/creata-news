package news

import (
	"context"
	"testing"
)

func TestOpportunityPrototypePhotoAndAudiencePersistAcrossCreatePatchAndPublicRead(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	input := Doc{"title": localized("A Creata opportunity"), "description": localized("Apply to build a project"), "organization": "Creata", "type": "grant", "url": "https://example.org/apply", "country": "KZ", "deadline": "2099-01-01T00:00:00Z", "status": "approved", "photo": "/news/photos/c35.jpg", "forWhom": Doc{"ru": "Студенты и создатели", "kk": "Студенттер мен жасаушылар", "en": "Students and creators"}}
	input["photoRights"] = Doc{"credit": "Creata reference artwork", "rights": "development fixture"}
	created := a.request("POST", "/admin/opportunities", input, 200, "")
	oid := str(created["id"])
	if str(created["photo"]) != "/news/photos/c35.jpg" || localize(created["forWhom"], "kk") != "Студенттер мен жасаушылар" {
		t.Fatalf("prototype fields were discarded on create: %v", created)
	}
	stored, e := a.store.Get(context.Background(), "opportunity", oid)
	if e != nil {
		t.Fatal(e)
	}
	if str(stored["photo"]) != str(created["photo"]) || localize(stored["forWhom"], "en") != "Students and creators" {
		t.Fatal("prototype fields were not persisted")
	}
	// A metadata-only edit must preserve rights/provenance already attached by an importer.
	if e = a.store.Write(context.Background(), func(tx *Tx) error {
		stored["photoRights"] = Doc{"credit": "Creata reference artwork", "rights": "development fixture"}
		return tx.Put(context.Background(), "opportunity", stored)
	}); e != nil {
		t.Fatal(e)
	}
	patched := a.request("PATCH", "/admin/opportunities/"+oid, Doc{"photo": "https://cdn.example.org/news/opportunity.webp", "forWhom": localized("Early-stage teams")}, 200, "")
	if str(patched["photo"]) != "https://cdn.example.org/news/opportunity.webp" || localize(patched["forWhom"], "ru") != "Early-stage teams" {
		t.Fatal("prototype fields were discarded on patch")
	}
	if str(object(patched["photoRights"])["credit"]) != "Creata reference artwork" {
		t.Fatal("existing photo rights metadata was lost")
	}
	public := a.request("GET", "/public/opportunities/"+oid, nil, 200, "")
	if str(public["photo"]) != str(patched["photo"]) || localize(public["forWhom"], "en") != "Early-stage teams" {
		t.Fatal("public detail omitted prototype fields")
	}
	list := array(a.request("GET", "/public/opportunities", nil, 200, "")["items"])
	if len(list) != 1 || str(object(list[0])["photo"]) != str(patched["photo"]) {
		t.Fatal("public list omitted prototype photo")
	}
	a.request("PATCH", "/admin/opportunities/"+oid, Doc{"organization": "Renamed organization"}, 200, "")
	kept := a.request("GET", "/public/opportunities/"+oid, nil, 200, "")
	if str(kept["photo"]) != str(patched["photo"]) || localize(kept["forWhom"], "kk") != "Early-stage teams" {
		t.Fatal("unrelated edit erased optional fields")
	}
}

func TestOpportunityOptionalPrototypeFieldsRemainBackwardCompatibleAndValidated(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	created := a.request("POST", "/admin/opportunities", Doc{"title": localized("An existing opportunity"), "description": localized("Existing content"), "organization": "Creata", "type": "course", "url": "https://example.org/course", "country": "KZ", "deadline": "2099-01-01T00:00:00Z", "status": "approved"}, 200, "")
	oid := str(created["id"])
	a.request("GET", "/public/opportunities/"+oid, nil, 200, "")
	for _, photo := range []any{"javascript:alert(1)", "//external.example/image.jpg", "/\\external.example/image.jpg", []any{"bad"}} {
		a.request("PATCH", "/admin/opportunities/"+oid, Doc{"photo": photo}, 400, "")
	}
	a.request("PATCH", "/admin/opportunities/"+oid, Doc{"forWhom": "plain unsupported value"}, 400, "")
	a.request("PATCH", "/admin/opportunities/"+oid, Doc{"forWhom": Doc{"ru": "Only one locale"}}, 400, "")
	a.request("PATCH", "/admin/opportunities/"+oid, Doc{"photo": "", "forWhom": localized("")}, 200, "")
}
