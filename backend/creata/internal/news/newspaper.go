package news

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"
)

func newspaperPublisher(u Doc) bool {
	return has(u["roles"], "chief_editor") || has(u["roles"], "admin")
}
func newspaperEditor(u, d Doc) bool {
	return newspaperPublisher(u) || has(u["roles"], "editor") || str(u["id"]) == str(d["creatorId"])
}

func newspaperMedia(value any) (Doc, error) {
	m := object(value)
	out := Doc{}
	for _, key := range []string{"url", "caption", "credit", "rights"} {
		v := str(m[key])
		if len(v) > 4000 {
			return nil, errStatus(400, "Photo details are too long")
		}
		out[key] = v
	}
	if str(out["url"]) != "" {
		if e := validateOpportunityPrototypeFields(Doc{"photo": out["url"]}); e != nil {
			return nil, e
		}
		if strings.TrimSpace(str(out["credit"])) == "" || strings.TrimSpace(str(out["rights"])) == "" {
			return nil, errStatus(400, "Photo credit and usage rights are required")
		}
	}
	return out, nil
}

func prepareNewspaper(ctx context.Context, tx *Tx, out, input Doc, editing bool) error {
	if editing {
		if number(input["expectedRevision"]) != float64(integer(out["revision"])) {
			return errStatus(409, "Issue changed; reload before saving")
		}
		if slug, ok := input["slug"]; ok && str(slug) != str(out["slug"]) {
			return errStatus(400, "An issue URL cannot change after creation")
		}
	}
	for _, key := range []string{"slug", "number", "date", "title", "description", "locale", "cover", "items"} {
		if v, ok := input[key]; ok {
			out[key] = v
		}
	}
	if !launchSlug.MatchString(str(out["slug"])) || len(str(out["slug"])) > 80 {
		return errStatus(400, "Use a short issue URL with lowercase letters, numbers and hyphens")
	}
	if str(out["slug"]) == "01" {
		return errStatus(409, "Original issue 01 is managed in Site content, including while unpublished")
	}
	for _, key := range []string{"number", "title"} {
		if strings.TrimSpace(str(out[key])) == "" || len(str(out[key])) > 240 {
			return errStatus(400, "Issue number and title are required (up to 240 characters)")
		}
	}
	if _, e := time.Parse("2006-01", str(out["date"])); e != nil {
		return errStatus(400, "Choose a valid issue month")
	}
	if !has([]string{"ru", "kk", "en"}, str(out["locale"])) {
		return errStatus(400, "Choose an issue language")
	}
	if len(str(out["description"])) > 8000 {
		return errStatus(400, "Issue introduction is too long")
	}
	cover, e := newspaperMedia(out["cover"])
	if e != nil {
		return e
	}
	out["cover"] = cover
	items := array(out["items"])
	if len(items) > 80 {
		return errStatus(400, "An issue can contain up to 80 articles")
	}
	seenIDs, seenArticles := map[string]bool{}, map[string]bool{}
	clean := []any{}
	for _, raw := range items {
		item := object(raw)
		key, articleID := str(item["id"]), str(item["articleId"])
		if key == "" || len(key) > 100 || articleID == "" || seenIDs[key] || seenArticles[articleID] {
			return errStatus(400, "Select each article once and use unique page identifiers")
		}
		if _, e := tx.Get(ctx, "article", articleID); e != nil {
			return errStatus(400, "A selected article is no longer available")
		}
		seenIDs[key], seenArticles[articleID] = true, true
		entry := Doc{"id": key, "articleId": articleID}
		if item["photo"] != nil {
			photo, e := newspaperMedia(item["photo"])
			if e != nil {
				return e
			}
			entry["photo"] = photo
		}
		clean = append(clean, entry)
	}
	out["items"] = clean
	all, e := tx.List(ctx, "newspaper_issue")
	if e != nil {
		return e
	}
	for _, d := range all {
		if str(d["id"]) != str(out["id"]) && str(d["slug"]) == str(out["slug"]) {
			return errStatus(409, "This issue URL is already in use")
		}
	}
	publications, e := tx.List(ctx, "newspaper_publication")
	if e != nil {
		return e
	}
	for _, p := range publications {
		if str(p["id"]) != str(out["id"]) && str(p["slug"]) == str(out["slug"]) {
			return errStatus(409, "This issue URL is already published")
		}
	}
	out["revision"] = integer(out["revision"]) + 1
	out["status"] = "draft"
	out["updatedAt"] = now()
	return nil
}

// Copy the public article version into an edition. Later article changes never
// replace the text of a previously published newspaper implicitly.
func newspaperSnapshot(ctx context.Context, tx *Tx, d Doc, publishing bool) (Doc, error) {
	p := clone(d)
	delete(p, "creatorId")
	delete(p, "items")
	entries := []any{}
	for _, raw := range array(d["items"]) {
		item := object(raw)
		article, e := tx.Get(ctx, "publication", str(item["articleId"]))
		if e != nil {
			return nil, errStatus(400, "Publish every selected article before publishing or previewing the issue")
		}
		lang := str(d["locale"])
		if str(object(article["title"])[lang]) == "" || str(object(article["body"])[lang]) == "" {
			return nil, errStatus(400, "A selected article has no published version in the issue language")
		}
		photo := object(item["photo"])
		if str(photo["url"]) == "" {
			photo = object(article["media"])
		}
		photo, e = newspaperMedia(photo)
		if e != nil {
			return nil, e
		}
		snapshot := publicArticle(article)
		if author, e := tx.Get(ctx, "author", str(article["authorId"])); e == nil {
			snapshot["author"] = author
		}
		entries = append(entries, Doc{"id": item["id"], "articleId": item["articleId"], "article": snapshot, "photo": photo})
	}
	if publishing && len(entries) == 0 {
		return nil, errStatus(400, "Choose at least one published article")
	}
	cover := object(d["cover"])
	if str(cover["url"]) == "" && len(entries) > 0 {
		cover = object(object(entries[0])["photo"])
	}
	if publishing && str(cover["url"]) == "" {
		return nil, errStatus(400, "Upload a cover photo or add a photo to the first article")
	}
	p["cover"] = cover
	p["entries"] = entries
	p["pages"] = len(entries)
	p["status"] = "published"
	p["publishedRevision"] = d["revision"]
	return p, nil
}

func (s *Server) newspaperRoutes() {
	s.handle("GET "+base+"/admin/newspapers", func(w http.ResponseWriter, r *http.Request) error {
		if _, e := s.require(r, true); e != nil {
			return e
		}
		return s.listResponse(w, r, "newspaper_issue")
	})
	s.handle("POST "+base+"/admin/newspapers", s.newspaperSave)
	s.handle("GET "+base+"/admin/newspapers/{id}", func(w http.ResponseWriter, r *http.Request) error {
		if _, e := s.require(r, true); e != nil {
			return e
		}
		d, e := s.Store.Get(r.Context(), "newspaper_issue", r.PathValue("id"))
		if e != nil {
			return e
		}
		respond(w, 200, d)
		return nil
	})
	s.handle("PATCH "+base+"/admin/newspapers/{id}", s.newspaperSave)
	s.handle("POST "+base+"/admin/newspapers/{id}/publish", s.newspaperPublish)
	s.handle("POST "+base+"/admin/newspapers/{id}/unpublish", s.newspaperPublish)
	s.handle("GET "+base+"/admin/newspapers/{id}/preview", func(w http.ResponseWriter, r *http.Request) error {
		if _, e := s.require(r, true); e != nil {
			return e
		}
		var p Doc
		e := s.Store.Write(r.Context(), func(tx *Tx) error {
			d, e := tx.Get(r.Context(), "newspaper_issue", r.PathValue("id"))
			if e != nil {
				return e
			}
			p, e = newspaperSnapshot(r.Context(), tx, d, false)
			return e
		})
		if e != nil {
			return e
		}
		respond(w, 200, p)
		return nil
	})
	s.handle("GET "+base+"/public/newspapers", func(w http.ResponseWriter, r *http.Request) error {
		items, e := s.Store.List(r.Context(), "newspaper_publication")
		if e != nil {
			return e
		}
		sort.SliceStable(items, func(i, j int) bool {
			if str(items[i]["date"]) == str(items[j]["date"]) {
				return str(items[i]["publishedAt"]) > str(items[j]["publishedAt"])
			}
			return str(items[i]["date"]) > str(items[j]["date"])
		})
		for _, d := range items {
			delete(d, "html")
		}
		respond(w, 200, Doc{"items": items, "total": len(items)})
		return nil
	})
	s.handle("GET "+base+"/public/newspapers/{slug}", func(w http.ResponseWriter, r *http.Request) error {
		items, e := s.Store.List(r.Context(), "newspaper_publication")
		if e != nil {
			return e
		}
		for _, d := range items {
			if str(d["slug"]) == r.PathValue("slug") {
				respond(w, 200, d)
				return nil
			}
		}
		return errStatus(404, "Issue not found")
	})
}

func (s *Server) newspaperSave(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	input, e := decode(r)
	if e != nil {
		return e
	}
	editing := r.PathValue("id") != ""
	var out Doc
	e = s.Store.Write(r.Context(), func(tx *Tx) error {
		if editing {
			var e error
			out, e = tx.Get(r.Context(), "newspaper_issue", r.PathValue("id"))
			if e != nil {
				return e
			}
			if !newspaperEditor(u, out) {
				return errStatus(403, "Only the issue author or an editor may edit this draft")
			}
		} else {
			out = Doc{"id": id(), "creatorId": u["id"], "revision": 0, "createdAt": now(), "items": []any{}, "description": ""}
		}
		if e := prepareNewspaper(r.Context(), tx, out, input, editing); e != nil {
			return e
		}
		if e := tx.Put(r.Context(), "newspaper_issue", out); e != nil {
			return e
		}
		return tx.Audit(r.Context(), u, "newspaper.save", str(out["id"]), Doc{"revision": out["revision"]})
	})
	if e != nil {
		return e
	}
	status := 201
	if editing {
		status = 200
	}
	respond(w, status, out)
	return nil
}

func (s *Server) newspaperPublish(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !newspaperPublisher(u) {
		return errStatus(403, "Only a chief editor or administrator may publish a newspaper")
	}
	input, e := decode(r)
	if e != nil {
		return e
	}
	var out Doc
	e = s.Store.Write(r.Context(), func(tx *Tx) error {
		var e error
		out, e = tx.Get(r.Context(), "newspaper_issue", r.PathValue("id"))
		if e != nil {
			return e
		}
		if number(input["expectedRevision"]) != float64(integer(out["revision"])) {
			return errStatus(409, "Issue changed; reload before publishing")
		}
		action := "publish"
		if strings.HasSuffix(r.URL.Path, "/unpublish") {
			action = "unpublish"
			if e = tx.Delete(r.Context(), "newspaper_publication", str(out["id"])); e != nil {
				return e
			}
			out["status"] = "draft"
			delete(out, "publishedRevision")
			delete(out, "publishedAt")
		} else {
			if integer(out["publishedRevision"]) == integer(out["revision"]) && str(out["status"]) == "published" {
				return nil
			}
			p, e := newspaperSnapshot(r.Context(), tx, out, true)
			if e != nil {
				return e
			}
			p["publishedAt"] = now()
			if e = tx.Put(r.Context(), "newspaper_publication", p); e != nil {
				return e
			}
			out["status"] = "published"
			out["publishedRevision"] = out["revision"]
			out["publishedAt"] = p["publishedAt"]
		}
		out["updatedAt"] = now()
		if e = tx.Put(r.Context(), "newspaper_issue", out); e != nil {
			return e
		}
		return tx.Audit(r.Context(), u, "newspaper."+action, str(out["id"]), Doc{"revision": out["revision"]})
	})
	if e != nil {
		return e
	}
	respond(w, 200, out)
	return nil
}
