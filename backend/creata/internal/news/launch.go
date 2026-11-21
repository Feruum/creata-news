package news

import (
	"context"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

var launchSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func validateLaunch(d Doc) error {
	if !launchSlug.MatchString(str(d["slug"])) || len(str(d["slug"])) > 100 {
		return errStatus(400, "Use a slug containing lowercase letters, numbers and hyphens")
	}
	if strings.TrimSpace(str(d["name"])) == "" || len(str(d["name"])) > 200 {
		return errStatus(400, "A startup name is required (up to 200 characters)")
	}
	for _, key := range []string{"country", "categoryId"} {
		if _, ok := d[key].(string); !ok || len(str(d[key])) > 100 {
			return errStatus(400, key+" must be a string (up to 100 characters)")
		}
	}
	if _, e := canonicalURL(str(d["url"])); e != nil {
		return errStatus(400, "A valid product URL is required")
	}
	if _, e := time.Parse("2006-01-02", str(d["launchDate"])); e != nil {
		return errStatus(400, "A valid launch date is required")
	}
	if !has([]string{"draft", "published", "archived"}, str(d["status"])) {
		return errStatus(400, "Invalid launch status")
	}
	for _, field := range []string{"tagline", "description", "team"} {
		text := object(d[field])
		filled := false
		for _, lang := range []string{"ru", "kk", "en"} {
			value, ok := text[lang].(string)
			if !ok || len(value) > 20000 {
				return errStatus(400, field+" must contain ru, kk and en strings (up to 20000 characters)")
			}
			filled = filled || strings.TrimSpace(value) != ""
		}
		if !filled {
			return errStatus(400, field+" is required in at least one language")
		}
	}
	if tags, present := d["tags"]; present {
		list, ok := tags.([]any)
		if !ok || len(list) > 8 {
			return errStatus(400, "Use up to eight tags")
		}
		for _, tag := range list {
			value, ok := tag.(string)
			if !ok || strings.TrimSpace(value) == "" || len(value) > 80 {
				return errStatus(400, "Tags must be nonempty strings (up to 80 characters)")
			}
		}
	}
	if logo, present := d["logoUrl"]; present {
		if e := validateOpportunityPrototypeFields(Doc{"photo": logo}); e != nil {
			return e
		}
	}
	return validateOpportunityPrototypeFields(d)
}

func prepareLaunch(ctx context.Context, tx *Tx, out, input Doc, editing bool) error {
	if editing {
		if integer(input["expectedRevision"]) != integer(out["revision"]) {
			return errStatus(409, "Startup changed; reload before saving")
		}
		if slug, ok := input["slug"]; ok {
			if value, valid := slug.(string); !valid || value != str(out["slug"]) {
				return errStatus(400, "A startup URL cannot change after creation")
			}
		}
	} else {
		out["status"] = "draft"
		out["revision"] = 0
	}
	for _, key := range []string{"slug", "name", "tagline", "description", "team", "url", "launchDate", "country", "categoryId", "photo", "logoUrl", "tags", "status"} {
		if value, ok := input[key]; ok {
			out[key] = value
		}
	}
	if e := validateLaunch(out); e != nil {
		return e
	}
	if str(out["categoryId"]) != "" {
		if _, e := tx.Get(ctx, "category", str(out["categoryId"])); e != nil {
			return errStatus(400, "Choose an existing category")
		}
	}
	all, e := tx.List(ctx, "launch")
	if e != nil {
		return e
	}
	for _, item := range all {
		if str(item["id"]) != str(out["id"]) && str(item["slug"]) == str(out["slug"]) {
			return errStatus(409, "This startup URL is already in use")
		}
	}
	out["revision"] = integer(out["revision"]) + 1
	return nil
}

func (s *Server) launchRoutes() {
	s.launchSocialRoutes()
	s.handle("GET "+base+"/admin/launches", func(w http.ResponseWriter, r *http.Request) error {
		if _, e := s.require(r, true); e != nil {
			return e
		}
		return s.listResponse(w, r, "launch")
	})
	s.handle("POST "+base+"/admin/launches", func(w http.ResponseWriter, r *http.Request) error { return s.simpleEdit(w, r, "launch") })
	s.handle("PATCH "+base+"/admin/launches/{id}", func(w http.ResponseWriter, r *http.Request) error { return s.simpleEdit(w, r, "launch") })
	s.handle("GET "+base+"/public/launches", s.launchesPublic)
	s.handle("GET "+base+"/public/launches/{slug}", s.launchPublic)
}

func (s *Server) launchesPublic(w http.ResponseWriter, r *http.Request) error {
	all, e := s.Store.List(r.Context(), "launch")
	if e != nil {
		return e
	}
	q := r.URL.Query()
	items := []Doc{}
	for _, d := range all {
		if str(d["status"]) != "published" {
			continue
		}
		if q.Get("category") != "" && q.Get("category") != str(d["categoryId"]) {
			continue
		}
		if q.Get("country") != "" && !strings.EqualFold(q.Get("country"), str(d["country"])) {
			continue
		}
		text := str(d["name"]) + " " + localize(d["tagline"], locale(q.Get("locale"))) + " " + localize(d["description"], locale(q.Get("locale")))
		for _, tag := range array(d["tags"]) {
			text += " " + str(tag)
		}
		if !strings.Contains(strings.ToLower(text), strings.ToLower(q.Get("q"))) {
			continue
		}
		items = append(items, d)
	}
	if e = s.launchStats(r.Context(), items); e != nil {
		return e
	}
	sort.SliceStable(items, func(i, j int) bool { return str(items[i]["launchDate"]) > str(items[j]["launchDate"]) })
	respond(w, 200, Doc{"items": items, "total": len(items)})
	return nil
}

func (s *Server) launchPublic(w http.ResponseWriter, r *http.Request) error {
	items, e := s.Store.List(r.Context(), "launch")
	if e != nil {
		return e
	}
	for _, d := range items {
		if str(d["slug"]) == r.PathValue("slug") && str(d["status"]) == "published" {
			if e = s.launchStats(r.Context(), []Doc{d}); e != nil {
				return e
			}
			respond(w, 200, d)
			return nil
		}
	}
	return errStatus(404, "Startup not found")
}
