package news

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func sitePageSlug(slug string) bool {
	return slug == "about" || slug == "ecosystem" || slug == "newspaper-01"
}
func nonempty(v any) bool { return strings.TrimSpace(str(v)) != "" }
func localizedRequired(v any) bool {
	d := object(v)
	for _, l := range []string{"ru", "kk", "en"} {
		if !nonempty(d[l]) {
			return false
		}
	}
	return true
}
func contentURL(v any) bool {
	s := str(v)
	if strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "//") && !strings.ContainsAny(s, "\\\r\n") {
		return true
	}
	u, e := url.Parse(s)
	return e == nil && ((u.Scheme == "http" || u.Scheme == "https") && u.Host != "" || u.Scheme == "mailto" && u.Opaque != "")
}

func validateOriginalHTML(value string) error {
	if len(value) == 0 || len(value) > 1500000 {
		return errStatus(400, "Original issue HTML is required and must be smaller than 1.5 MB")
	}
	doc, e := html.Parse(strings.NewReader(value))
	if e != nil {
		return errStatus(400, "Invalid issue HTML")
	}
	var walk func(*html.Node) error
	walk = func(n *html.Node) error {
		if n.Type == html.ElementNode {
			if has([]string{"script", "style", "iframe", "object", "embed", "base", "link", "meta", "form", "input", "textarea", "foreignObject", "foreignobject", "animate", "set"}, n.Data) {
				return errStatus(400, "Active HTML is forbidden in an archived issue")
			}
			for _, a := range n.Attr {
				key := strings.ToLower(a.Key)
				v := strings.ToLower(strings.TrimSpace(a.Val))
				if strings.HasPrefix(key, "on") || key == "srcdoc" || key == "style" && (strings.Contains(v, "url(") || strings.Contains(v, "expression(")) {
					return errStatus(400, "Active HTML attributes are forbidden")
				}
				if has([]string{"href", "src", "poster", "data-loop", "xlink:href"}, key) && a.Val != "" && !strings.HasPrefix(a.Val, "#") && !contentURL(a.Val) {
					return errStatus(400, "Unsafe issue link")
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if e := walk(c); e != nil {
				return e
			}
		}
		return nil
	}
	return walk(doc)
}

func validateSiteContent(slug string, c Doc) error {
	if !sitePageSlug(slug) || len(c) == 0 {
		return errStatus(400, "Unknown or empty site page")
	}
	if slug == "about" {
		for _, l := range []string{"ru", "kk", "en"} {
			d := object(c[l])
			if !nonempty(d["title"]) || !nonempty(d["body"]) || !nonempty(d["second"]) || len(array(d["values"])) == 0 {
				return errStatus(400, "About page requires complete RU/ҚАЗ/EN text")
			}
			for _, v := range array(d["values"]) {
				p := array(v)
				if len(p) != 2 || !nonempty(p[0]) || !nonempty(p[1]) {
					return errStatus(400, "Each value needs a title and description")
				}
			}
		}
	}
	if slug == "ecosystem" {
		if !launchSlug.MatchString(str(c["issueSlug"])) || !nonempty(c["issueNumber"]) {
			return errStatus(400, "Choose a valid linked issue slug and number")
		}
		for _, l := range []string{"ru", "kk", "en"} {
			d := object(object(c["copy"])[l])
			for _, k := range []string{"title", "description", "eyebrow", "intro", "explore", "meet", "photoCredit", "who", "whoTitle", "whoBody", "whoSecond", "group", "companies", "layers", "route", "model", "modelTitle", "modelLead", "products", "productsTitle", "productsLead", "openProduct", "stories", "storiesTitle", "issueTitle", "issueBody", "readIssue", "next", "nextTitle", "nextBody", "opportunities", "contact", "backToProducts"} {
				if !nonempty(d[k]) {
					return errStatus(400, "Missing ecosystem text: "+l+"."+k)
				}
			}
			if len(array(d["hero"])) == 0 {
				return errStatus(400, "Ecosystem hero text is required")
			}
		}
		for _, k := range []string{"photos", "layers", "products"} {
			if len(array(c[k])) == 0 {
				return errStatus(400, "Ecosystem requires photos, layers and products")
			}
		}
		for _, v := range array(c["photos"]) {
			d := object(v)
			if !contentURL(d["src"]) || !nonempty(d["label"]) || !localizedRequired(d["caption"]) {
				return errStatus(400, "Complete each gallery photograph")
			}
		}
		for _, v := range array(c["layers"]) {
			d := object(v)
			for _, k := range []string{"title", "subtitle", "description", "caption"} {
				if !localizedRequired(d[k]) {
					return errStatus(400, "Complete localized layer text")
				}
			}
			if !nonempty(d["id"]) || !nonempty(d["number"]) || !contentURL(d["photo"]) || len(array(d["companies"])) == 0 {
				return errStatus(400, "Complete ecosystem layer")
			}
			for _, v := range array(d["companies"]) {
				co := object(v)
				if !nonempty(co["name"]) || !contentURL(co["href"]) || !localizedRequired(co["note"]) {
					return errStatus(400, "Complete each ecosystem company")
				}
			}
		}
		seen := map[string]bool{}
		for _, v := range array(c["products"]) {
			d := object(v)
			key := str(d["id"])
			if key == "" || seen[key] || !nonempty(d["name"]) || !contentURL(d["href"]) || !localizedRequired(d["audience"]) || !localizedRequired(d["description"]) {
				return errStatus(400, "Complete each product and use unique IDs")
			}
			seen[key] = true
		}
		for _, k := range []string{"issueCover", "groupUrl", "contactUrl"} {
			if !contentURL(c[k]) {
				return errStatus(400, "Invalid ecosystem link: "+k)
			}
		}
	}
	if slug == "newspaper-01" {
		if str(c["slug"]) != "01" || !nonempty(c["title"]) || !nonempty(c["number"]) || integer(c["pages"]) < 1 || !has([]string{"ru", "kk", "en"}, str(c["locale"])) {
			return errStatus(400, "Complete original newspaper metadata")
		}
		if _, e := time.Parse("2006-01", str(c["date"])); e != nil {
			return errStatus(400, "Invalid issue month")
		}
		if !contentURL(object(c["cover"])["url"]) {
			return errStatus(400, "Cover image is required")
		}
		if str(c["pdf"]) != "" && !contentURL(c["pdf"]) {
			return errStatus(400, "Invalid PDF URL")
		}
		return validateOriginalHTML(str(c["html"]))
	}
	return nil
}

func putPagePublication(ctx context.Context, t *Tx, d Doc) error {
	p := clone(d)
	p["status"] = "published"
	p["publishedRevision"] = d["revision"]
	p["publishedAt"] = now()
	if e := t.Put(ctx, "site_page_publication", p); e != nil {
		return e
	}
	if str(d["id"]) == "newspaper-01" {
		n := clone(object(d["content"]))
		n["id"] = "original-01"
		n["format"] = "original"
		n["status"] = "published"
		n["publishedRevision"] = d["revision"]
		n["revision"] = d["revision"]
		n["publishedAt"] = p["publishedAt"]
		n["entries"] = []any{}
		cover, e := newspaperMedia(n["cover"])
		if e != nil {
			return e
		}
		n["cover"] = cover
		return t.Put(ctx, "newspaper_publication", n)
	}
	return nil
}

func (s *Server) contentRoutes() {
	s.handle("GET "+base+"/public/pages/{slug}", func(w http.ResponseWriter, r *http.Request) error {
		d, e := s.Store.Get(r.Context(), "site_page_publication", r.PathValue("slug"))
		if e != nil {
			return e
		}
		respond(w, 200, d)
		return nil
	})
	s.handle("GET "+base+"/admin/pages", func(w http.ResponseWriter, r *http.Request) error {
		if _, e := s.require(r, true); e != nil {
			return e
		}
		return s.listResponse(w, r, "site_page")
	})
	s.handle("GET "+base+"/admin/pages/{slug}", func(w http.ResponseWriter, r *http.Request) error {
		if _, e := s.require(r, true); e != nil {
			return e
		}
		d, e := s.Store.Get(r.Context(), "site_page", r.PathValue("slug"))
		if e != nil {
			return e
		}
		respond(w, 200, d)
		return nil
	})
	s.handle("GET "+base+"/admin/pages/{slug}/revisions", func(w http.ResponseWriter, r *http.Request) error {
		if _, e := s.require(r, true); e != nil {
			return e
		}
		all, e := s.Store.List(r.Context(), "site_page_revision")
		if e != nil {
			return e
		}
		items := []Doc{}
		for _, d := range all {
			if str(d["slug"]) == r.PathValue("slug") {
				items = append(items, d)
			}
		}
		respond(w, 200, Doc{"items": items, "total": len(items)})
		return nil
	})
	s.handle("PATCH "+base+"/admin/pages/{slug}", s.saveSitePage)
	s.handle("POST "+base+"/admin/pages/{slug}/publish", s.publishSitePage)
	s.handle("POST "+base+"/admin/pages/{slug}/unpublish", s.publishSitePage)
}
func (s *Server) saveSitePage(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !has(u["roles"], "editor") && !newspaperPublisher(u) {
		return errStatus(403, "Only an editor can edit site content")
	}
	in, e := decode(r)
	if e != nil {
		return e
	}
	var out Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		d, e := t.Get(r.Context(), "site_page", r.PathValue("slug"))
		if e != nil {
			return e
		}
		if integer(in["expectedRevision"]) != integer(d["revision"]) {
			return errStatus(409, "Page changed; reload before saving")
		}
		c := object(in["content"])
		if e = validateSiteContent(str(d["id"]), c); e != nil {
			return e
		}
		d["content"] = c
		d["revision"] = integer(d["revision"]) + 1
		d["status"] = "draft"
		d["updatedAt"] = now()
		d["actorId"] = u["id"]
		out = d
		if e = t.Put(r.Context(), "site_page", d); e != nil {
			return e
		}
		rev := clone(d)
		rev["slug"] = d["id"]
		rev["id"] = fmt.Sprintf("%s:%d", d["id"], integer(d["revision"]))
		if e = t.Put(r.Context(), "site_page_revision", rev); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "page.save", str(d["id"]), Doc{"revision": d["revision"]})
	})
	if e != nil {
		return e
	}
	respond(w, 200, out)
	return nil
}
func (s *Server) publishSitePage(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !newspaperPublisher(u) {
		return errStatus(403, "Only a chief editor or administrator may publish site content")
	}
	in, e := decode(r)
	if e != nil {
		return e
	}
	var out Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		d, e := t.Get(r.Context(), "site_page", r.PathValue("slug"))
		if e != nil {
			return e
		}
		if integer(in["expectedRevision"]) != integer(d["revision"]) {
			return errStatus(409, "Page changed; reload before publishing")
		}
		action := "publish"
		if strings.HasSuffix(r.URL.Path, "/unpublish") {
			action = "unpublish"
			if e = t.Delete(r.Context(), "site_page_publication", str(d["id"])); e != nil {
				return e
			}
			if str(d["id"]) == "newspaper-01" {
				if e = t.Delete(r.Context(), "newspaper_publication", "original-01"); e != nil {
					return e
				}
			}
			d["status"] = "draft"
			delete(d, "publishedRevision")
		} else {
			if e = validateSiteContent(str(d["id"]), object(d["content"])); e != nil {
				return e
			}
			if e = putPagePublication(r.Context(), t, d); e != nil {
				return e
			}
			d["status"] = "published"
			d["publishedRevision"] = d["revision"]
		}
		out = d
		if e = t.Put(r.Context(), "site_page", d); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "page."+action, str(d["id"]), Doc{"revision": d["revision"]})
	})
	if e != nil {
		return e
	}
	respond(w, 200, out)
	return nil
}

// ImportContent migrates existing approved content exactly once; it never overwrites editorial revisions.
func (s *Server) ImportContent(ctx context.Context, bundle Doc, assetDir string) error {
	b, e := json.Marshal(bundle)
	if e != nil {
		return e
	}
	if assetDir != "" {
		b, e = s.importContentAssets(ctx, b, assetDir)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(b, &bundle); e != nil {
			return e
		}
	}
	return s.Store.Write(ctx, func(t *Tx) error {
		for _, v := range array(bundle["pages"]) {
			page := object(v)
			slug := str(page["slug"])
			if existing, e := t.Get(ctx, "site_page", slug); e == nil {
				if slug == "ecosystem" {
					if e = upgradeEcosystemIssueMetadata(ctx, t, existing, object(page["content"])); e != nil {
						return e
					}
				}
				continue
			}
			c := object(page["content"])
			if e := validateSiteContent(slug, c); e != nil {
				return fmt.Errorf("%s: %w", slug, e)
			}
			d := Doc{"id": slug, "slug": slug, "content": c, "revision": 1, "status": "published", "publishedRevision": 1, "createdAt": now(), "updatedAt": now(), "actorId": "migration"}
			if e := t.Put(ctx, "site_page", d); e != nil {
				return e
			}
			rev := clone(d)
			rev["slug"] = slug
			rev["id"] = slug + ":1"
			if e := t.Put(ctx, "site_page_revision", rev); e != nil {
				return e
			}
			if e := putPagePublication(ctx, t, d); e != nil {
				return e
			}
			if e := t.Audit(ctx, Doc{"id": "migration"}, "page.import", slug, Doc{"revision": 1}); e != nil {
				return e
			}
		}
		return nil
	})
}

// Old rendering displayed the original issue as a literal. Migrating that
// relationship into content preserves every existing draft/public revision.
func upgradeEcosystemIssueMetadata(ctx context.Context, t *Tx, current, defaults Doc) error {
	if !launchSlug.MatchString(str(defaults["issueSlug"])) || !nonempty(defaults["issueNumber"]) {
		return errStatus(400, "Missing issue relationship in backend import package")
	}
	upgrade := func(kind string, d Doc) error {
		c := object(d["content"])
		changed := false
		for _, key := range []string{"issueSlug", "issueNumber"} {
			if _, exists := c[key]; !exists {
				c[key] = defaults[key]
				changed = true
			}
		}
		if !changed {
			return nil
		}
		d["content"] = c
		return t.Put(ctx, kind, d)
	}
	if e := upgrade("site_page", current); e != nil {
		return e
	}
	if p, e := t.Get(ctx, "site_page_publication", "ecosystem"); e == nil {
		if e = upgrade("site_page_publication", p); e != nil {
			return e
		}
	}
	revisions, e := t.List(ctx, "site_page_revision")
	if e != nil {
		return e
	}
	for _, d := range revisions {
		if str(d["slug"]) == "ecosystem" {
			if e = upgrade("site_page_revision", d); e != nil {
				return e
			}
		}
	}
	return nil
}
