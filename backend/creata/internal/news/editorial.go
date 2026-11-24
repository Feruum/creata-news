package news

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (s *Server) editorialRoutes() {
	s.handle("GET "+base+"/admin/postiz/channels", s.postizChannels)
	s.handle("GET "+base+"/admin/articles/{id}/postiz", s.postizHistory)
	s.handle("GET "+base+"/admin/articles/{id}/postiz/preview", s.postizPreview)
	s.handle("GET "+base+"/admin/staff", s.staffDirectory)
	s.handle("GET "+base+"/admin/articles", s.adminArticles)
	s.handle("POST "+base+"/admin/articles", s.createArticle)
	s.handle("GET "+base+"/admin/articles/{id}", s.getArticle)
	s.handle("PATCH "+base+"/admin/articles/{id}", s.patchArticle)
	s.handle("POST "+base+"/admin/articles/{id}/actions", s.articleAction)
	s.handle("GET "+base+"/admin/articles/{id}/revisions", s.revisions)
	s.handle("GET "+base+"/admin/articles/{id}/research", s.articleResearch)
	s.handle("GET "+base+"/admin/bank", s.bankList)
	s.handle("GET "+base+"/admin/bank/similar", s.bankSimilar)
	s.handle("GET "+base+"/admin/bank/merges", s.bankMerges)
	s.handle("GET "+base+"/admin/bank/{id}/evidence", s.bankEvidence)
	s.handle("POST "+base+"/admin/bank/{id}/merge", s.bankMerge)
	s.handle("POST "+base+"/admin/bank/merges/{id}/undo", s.bankUndo)
	s.handle("POST "+base+"/admin/bank/merges/{id}/split", s.bankSplit)
	s.handle("POST "+base+"/admin/bank/collect", s.manualCollect)
	s.handle("POST "+base+"/admin/bank/{id}/draft", s.bankDraft)
	s.handle("POST "+base+"/admin/bank/{id}/graphic", s.bankGraphic)
	s.handle("POST "+base+"/admin/articles/{id}/graphic-draft", s.graphicDraft)
	for path, kind := range map[string]string{"opportunities": "opportunity", "authors": "author"} {
		p, k := path, kind
		s.handle("GET "+base+"/admin/"+p, func(w http.ResponseWriter, r *http.Request) error {
			if _, e := s.require(r, true); e != nil {
				return e
			}
			return s.listResponse(w, r, k)
		})
		s.handle("POST "+base+"/admin/"+p, func(w http.ResponseWriter, r *http.Request) error { return s.simpleEdit(w, r, k) })
		s.handle("PATCH "+base+"/admin/"+p+"/{id}", func(w http.ResponseWriter, r *http.Request) error { return s.simpleEdit(w, r, k) })
	}
	s.handle("GET "+base+"/admin/users", s.users)
	s.handle("PATCH "+base+"/admin/users/{id}", s.editUser)
	s.handle("GET "+base+"/admin/analytics", s.analytics)
	s.handle("GET "+base+"/admin/audit", func(w http.ResponseWriter, r *http.Request) error {
		u, e := s.require(r, true)
		if e != nil {
			return e
		}
		if !has(u["roles"], "admin") && !has(u["roles"], "chief_editor") {
			return errStatus(403, "Audit access denied")
		}
		return s.listResponse(w, r, "audit")
	})
}
func (s *Server) adminArticles(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	items, e := s.Store.List(r.Context(), "article")
	if e != nil {
		return e
	}
	out := []Doc{}
	q := r.URL.Query()
	for _, a := range items {
		if q.Get("format") != "" && materialFormat(a) != q.Get("format") {
			continue
		}
		if !canEdit(u, a) && !has(u["roles"], "admin") {
			continue
		}
		if q.Get("status") != "" && str(a["status"]) != q.Get("status") {
			continue
		}
		if q.Get("category") != "" && str(a["categoryId"]) != q.Get("category") {
			continue
		}
		if q.Get("q") != "" && !strings.Contains(strings.ToLower(localize(a["title"], locale(q.Get("locale")))), strings.ToLower(q.Get("q"))) {
			continue
		}
		out = append(out, s.decorate(r.Context(), a))
	}
	respond(w, 200, Doc{"items": out, "total": len(out)})
	return nil
}
func newArticle(u, d Doc) Doc {
	a := Doc{"id": id(), "title": localized(""), "lead": localized(""), "body": localized(""), "summary": localized(""), "actions": localized(""), "layer": "world", "categoryId": "ai", "region": "", "tags": []any{}, "authorId": u["id"], "sources": []any{}, "reviewNotes": []any{}, "status": "new", "revision": 1, "createdAt": now(), "updatedAt": now()}
	applyArticle(a, d)
	if str(a["slug"]) == "" {
		a["slug"] = slugify(localize(a["title"], "en")) + "-" + str(a["id"])[:8]
	}
	return a
}
func applyArticle(a, d Doc) {
	for _, k := range []string{"title", "lead", "body", "summary", "actions", "layer", "categoryId", "region", "tags", "authorId", "editorId", "sources", "reviewNotes", "media", "photos", "slug", "format", "graphic", "opportunityIds", "storyUpdates"} {
		if v, ok := d[k]; ok {
			a[k] = v
		}
	}
	if guide, exists := d["actionGuide"]; exists {
		if guide == nil {
			delete(a, "actionGuide")
		} else {
			a["actionGuide"] = guide
		}
	}
}
func validateArticle(a Doc) error {
	if err := validateArticleExtras(a, false); err != nil {
		return errStatus(400, err.Error())
	}
	if format := str(a["format"]); format != "" && format != "article" && format != "graphic" {
		return errStatus(400, "Invalid material format")
	}
	if a["graphic"] != nil {
		if str(a["format"]) != "graphic" {
			return errStatus(400, "Chart data requires graphic format")
		}
		if e := validateGraphic(a["graphic"], false); e != nil {
			return e
		}
	}
	for _, key := range []string{"authorId", "categoryId", "slug", "layer"} {
		if _, ok := a[key].(string); !ok || str(a[key]) == "" {
			return errStatus(400, key+" must be a nonempty string")
		}
	}
	if v, exists := a["editorId"]; exists {
		if _, ok := v.(string); !ok {
			return errStatus(400, "editorId must be a string")
		}
	}
	if str(a["layer"]) != "world" && str(a["layer"]) != "eco" {
		return errStatus(400, "Invalid article layer")
	}
	for _, f := range []string{"title", "lead", "body", "summary", "actions"} {
		if len(object(a[f])) == 0 {
			return errStatus(400, f+" must be localized text")
		}
		for _, l := range []string{"ru", "kk", "en"} {
			if _, ok := object(a[f])[l].(string); !ok {
				return errStatus(400, f+" must contain string values for ru, kk and en")
			}
			if len(str(object(a[f])[l])) > 200000 {
				return errStatus(400, "Article text is too long")
			}
		}
	}
	if len(array(a["sources"])) > 50 {
		return errStatus(400, "Too many sources")
	}
	for _, raw := range array(a["sources"]) {
		if _, e := canonicalURL(str(object(raw)["url"])); e != nil {
			return errStatus(400, "Invalid source URL")
		}
	}
	if strings.ContainsAny(str(a["slug"]), "/?#") || str(a["slug"]) == "" {
		return errStatus(400, "Invalid article slug")
	}
	return nil
}
func (t *Tx) saveRevision(ctx context.Context, a, u Doc, note string) error {
	if _, err := t.Exec(ctx, "UPDATE creata_postiz_delivery SET status='cancelled',updated_at=now() WHERE article_id=$1 AND status='scheduled' AND (revision<>$2 OR $3='')", a["id"], integer(a["revision"]), str(a["scheduledAt"])); err != nil {
		return err
	}
	from := ""
	if previous, e := t.Get(ctx, "article", str(a["id"])); e == nil {
		from = str(previous["status"])
	}
	if e := recordStage(t, ctx, a, from, u); e != nil {
		return e
	}
	rev := clone(a)
	rev["articleId"] = a["id"]
	rev["id"] = str(a["id"]) + ":" + fmt.Sprint(integer(a["revision"]))
	rev["actorId"] = u["id"]
	rev["note"] = note
	rev["createdAt"] = now()
	if e := t.Put(ctx, "revision", rev); e != nil {
		return e
	}
	return t.Put(ctx, "article", a)
}
func (s *Server) createArticle(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	a := newArticle(u, d)
	if !canEdit(u, a) {
		return errStatus(403, "You cannot create for this author or vertical")
	}
	if e = validateArticle(a); e != nil {
		return e
	}
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		if e := validateEditor(t, r.Context(), a); e != nil {
			return e
		}
		if e := t.saveRevision(r.Context(), a, u, "Created"); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "article.create", str(a["id"]), Doc{"stageRecorded": true})
	})
	if e != nil {
		return e
	}
	respond(w, 201, a)
	return nil
}
func (s *Server) getArticle(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	a, e := s.Store.Get(r.Context(), "article", r.PathValue("id"))
	if e != nil {
		return e
	}
	if !canEdit(u, a) && !has(u["roles"], "admin") {
		return errStatus(403, "Article access denied")
	}
	respond(w, 200, s.decorate(r.Context(), a))
	return nil
}
func (s *Server) patchArticle(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	var a Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		a, e = t.Get(r.Context(), "article", r.PathValue("id"))
		if e != nil {
			return e
		}
		if !canEdit(u, a) {
			return errStatus(403, "Article edit denied")
		}
		if integer(d["expectedRevision"]) != integer(a["revision"]) {
			return errStatus(409, "Article changed; reload before saving")
		}
		oldAuthor, oldCat := str(a["authorId"]), str(a["categoryId"])
		applyArticle(a, d)
		if !canPublish(u, a) && (str(a["authorId"]) != oldAuthor || str(a["categoryId"]) != oldCat) {
			return errStatus(403, "Assignment change denied")
		}
		revise(a)
		if e = validateArticle(a); e != nil {
			return e
		}
		if e = t.validateStoryPermalink(r.Context(), a); e != nil {
			return e
		}
		if e = validateEditor(t, r.Context(), a); e != nil {
			return e
		}
		if e = t.saveRevision(r.Context(), a, u, str(d["note"])); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "article.edit", str(a["id"]), Doc{"revision": a["revision"], "stageRecorded": true})
	})
	if e != nil {
		return e
	}
	respond(w, 200, a)
	return nil
}

func validateEditor(t *Tx, ctx context.Context, a Doc) error {
	if _, e := t.Get(ctx, "author", str(a["authorId"])); e != nil {
		return errStatus(400, "Article author not found")
	}
	if _, e := t.Get(ctx, "category", str(a["categoryId"])); e != nil {
		return errStatus(400, "Article category not found")
	}
	if value, exists := a["opportunityIds"]; exists {
		if _, ok := classificationArray(value, 12); !ok {
			return errStatus(400, "Linked opportunities must be an array of strings")
		}
	}
	if len(array(a["opportunityIds"])) > 12 {
		return errStatus(400, "Too many linked opportunities")
	}
	seen := map[string]bool{}
	for _, raw := range array(a["opportunityIds"]) {
		oid := str(raw)
		if oid == "" || seen[oid] {
			return errStatus(400, "Invalid linked opportunity identifier")
		}
		seen[oid] = true
		if _, e := t.Get(ctx, "opportunity", oid); e != nil {
			return errStatus(400, "Linked opportunity not found")
		}
	}
	eid := str(a["editorId"])
	if eid == "" {
		return nil
	}
	u, e := t.Get(ctx, "user", eid)
	if e != nil {
		return errStatus(400, "Responsible editor not found")
	}
	if !has(u["roles"], "chief_editor") && !(has(u["roles"], "editor") && has(u["verticals"], str(a["categoryId"]))) {
		return errStatus(400, "Responsible editor does not manage this vertical")
	}
	return nil
}
func (s *Server) staffDirectory(w http.ResponseWriter, r *http.Request) error {
	if _, e := s.require(r, true); e != nil {
		return e
	}
	users, e := s.Store.List(r.Context(), "user")
	if e != nil {
		return e
	}
	out := []Doc{}
	for _, u := range users {
		if has(u["roles"], "editor") || has(u["roles"], "chief_editor") {
			out = append(out, Doc{"id": u["id"], "name": u["name"], "roles": u["roles"], "verticals": u["verticals"]})
		}
	}
	respond(w, 200, Doc{"items": out})
	return nil
}
func (s *Server) articleAction(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	targets, e := s.postizTargets(r.Context(), u, r.PathValue("id"), d)
	if e != nil {
		return e
	}
	var a Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		a, e = t.Get(r.Context(), "article", r.PathValue("id"))
		if e != nil {
			return e
		}
		if !canEdit(u, a) {
			return errStatus(403, "Article access denied")
		}
		if integer(d["expectedRevision"]) != integer(a["revision"]) {
			return errStatus(409, "Article changed; reload")
		}
		action := str(d["action"])
		fromStatus := str(a["status"])
		if has([]string{"approve", "publish", "schedule"}, action) {
			if e = validateEditor(t, r.Context(), a); e != nil {
				return e
			}
		}
		switch action {
		case "submit":
			a["status"] = "review"
			delete(a, "scheduledAt")
			delete(a, "approvedRevision")
		case "approve":
			if !canPublish(u, a) {
				return errStatus(403, "Publication approval denied")
			}
			if e = t.validateStoryPermalink(r.Context(), a); e != nil {
				return e
			}
			if e = publishable(a); e != nil {
				return errStatus(400, e.Error())
			}
			a["approvedRevision"] = a["revision"]
			a["status"] = "approved"
		case "publish", "schedule":
			if !canPublish(u, a) {
				return errStatus(403, "Publication denied")
			}
			if integer(a["approvedRevision"]) != integer(a["revision"]) {
				return errStatus(409, "Current revision must be approved")
			}
			if e = publishable(a); e != nil {
				return errStatus(400, e.Error())
			}
			if action == "schedule" {
				if e = t.validateStoryPermalink(r.Context(), a); e != nil {
					return e
				}
				due := parseTime(d["scheduledAt"])
				if !due.After(time.Now()) {
					return errStatus(400, "Choose a future publication time")
				}
				a["scheduledAt"] = due.UTC().Format(time.RFC3339)
				a["publicationNote"] = str(d["note"])
			}
			if e = s.stagePostiz(t, r.Context(), a, targets); e != nil {
				return e
			}
			if action == "publish" {
				if e = publishEditorial(t, r.Context(), a, str(d["note"])); e != nil {
					return e
				}
			}
		case "unpublish":
			if !canPublish(u, a) {
				return errStatus(403, "Publication withdrawal denied")
			}
			if e = t.Delete(r.Context(), "publication", str(a["id"])); e != nil {
				return e
			}
			if _, e = t.Exec(r.Context(), "UPDATE creata_postiz_delivery SET status='cancelled',updated_at=now() WHERE article_id=$1 AND status='queued'", a["id"]); e != nil {
				return e
			}
			a["status"] = "review"
			delete(a, "publishedRevision")
			delete(a, "scheduledAt")
			delete(a, "approvedRevision")
		case "reject":
			if !canPublish(u, a) {
				return errStatus(403, "Review denied")
			}
			a["status"] = "rejected"
			delete(a, "approvedRevision")
			delete(a, "scheduledAt")
		case "rework":
			if strings.TrimSpace(str(d["note"])) == "" {
				return errStatus(400, "Describe the requested changes")
			}
			a["status"] = "rework"
			delete(a, "approvedRevision")
			delete(a, "scheduledAt")
			job := newJob("rework", Doc{"article": clone(a), "expectedRevision": a["revision"], "note": d["note"]})
			a["activeAIJobId"] = job["id"]
			stories, err := t.List(r.Context(), "story")
			if err != nil {
				return err
			}
			for _, story := range stories {
				if str(story["articleId"]) == str(a["id"]) {
					object(job["payload"])["story"] = story
					break
				}
			}
			if e = enqueuePipelineJob(t, r.Context(), job); e != nil {
				return e
			}
		default:
			return errStatus(400, "Unknown article action")
		}
		if action != "rework" {
			delete(a, "activeAIJobId")
		}
		a["updatedAt"] = now()
		if str(a["scheduledAt"]) == "" && action != "publish" {
			if e = cancelScheduledPostiz(t, r.Context(), str(a["id"])); e != nil {
				return e
			}
		}
		if action != "publish" {
			if e = recordStage(t, r.Context(), a, fromStatus, u); e != nil {
				return e
			}
		}
		if e = t.Put(r.Context(), "article", a); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "article."+action, str(a["id"]), Doc{"revision": a["revision"], "note": d["note"], "stageRecorded": true})
	})
	if e != nil {
		return e
	}
	respond(w, 200, a)
	return nil
}
func publish(t *Tx, ctx context.Context, a Doc, note string) error {
	if e := validateEditor(t, ctx, a); e != nil {
		return e
	}
	if e := validateArticleExtras(a, true); e != nil {
		return errStatus(400, e.Error())
	}
	if e := t.validateStoryPermalink(ctx, a); e != nil {
		return e
	}
	if len(array(a["storyUpdates"])) > 0 {
		a["storySlug"] = a["slug"]
	}
	from := str(a["status"])
	delete(a, "activeAIJobId")
	a["status"] = "published"
	if e := recordStage(t, ctx, a, from, Doc{"id": "publication"}); e != nil {
		return e
	}
	a["publishedRevision"] = a["revision"]
	a["publishedAt"] = now()
	delete(a, "scheduledAt")
	p := clone(a)
	if e := t.Put(ctx, "publication", p); e != nil {
		return e
	}
	v := clone(p)
	v["id"] = str(a["id"]) + ":" + fmt.Sprint(integer(a["revision"]))
	v["articleId"] = a["id"]
	v["note"] = note
	return t.Put(ctx, "publication_version", v)
}
func (s *Server) revisions(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	a, e := s.Store.Get(r.Context(), "article", r.PathValue("id"))
	if e != nil {
		return e
	}
	if !canEdit(u, a) && !has(u["roles"], "admin") {
		return errStatus(403, "Revision access denied")
	}
	all, e := s.Store.List(r.Context(), "revision")
	if e != nil {
		return e
	}
	out := []Doc{}
	for _, v := range all {
		if str(v["articleId"]) == r.PathValue("id") {
			out = append(out, v)
		}
	}
	respond(w, 200, Doc{"items": out})
	return nil
}
func (s *Server) bankDraft(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	var a Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		story, e := t.Get(r.Context(), "story", r.PathValue("id"))
		if e != nil {
			return e
		}
		if str(story["mergedInto"]) != "" {
			return errStatus(409, "Story was merged; refresh the bank")
		}
		if aid := str(story["articleId"]); aid != "" {
			a, e = t.Get(r.Context(), "article", aid)
			if e != nil {
				return e
			}
			if !canEdit(u, a) {
				return errStatus(403, "Article access denied")
			}
			return nil
		}
		a = newArticle(u, Doc{"title": localized(str(story["title"])), "categoryId": story["categoryId"], "sources": story["sources"], "region": story["region"], "tags": story["tags"]})
		copyArticlePhotos(a, story)
		if str(a["categoryId"]) == "" {
			a["categoryId"] = "ai"
		}
		if !canEdit(u, a) {
			return errStatus(403, "Vertical access denied")
		}
		a["status"] = "writing"
		if e = validateEditor(t, r.Context(), a); e != nil {
			return e
		}
		if e = t.saveRevision(r.Context(), a, u, "Created from story"); e != nil {
			return e
		}
		story["articleId"] = a["id"]
		story["status"] = "in_progress"
		story["updatedAt"] = now()
		if e = t.Put(r.Context(), "story", story); e != nil {
			return e
		}
		return enqueuePipelineJob(t, r.Context(), newJob("draft", Doc{"article": a, "story": story, "expectedRevision": a["revision"]}))
	})
	if e != nil {
		return e
	}
	respond(w, 201, a)
	return nil
}
func (s *Server) simpleEdit(w http.ResponseWriter, r *http.Request, kind string) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !has(u["roles"], "chief_editor") && !has(u["roles"], "editor") && !has(u["roles"], "admin") {
		return errStatus(403, "Management access denied")
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	var out Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		if r.Method == "PATCH" {
			out, e = t.Get(r.Context(), kind, r.PathValue("id"))
			if e != nil {
				return e
			}
		} else {
			out = Doc{"id": id(), "createdAt": now()}
			if kind == "opportunity" {
				out["status"] = "pending"
			}
		}
		if kind == "launch" {
			if e = prepareLaunch(r.Context(), t, out, d, r.Method == "PATCH"); e != nil {
				return e
			}
		}
		allowed := []string{"name", "bio", "avatarUrl", "role"}
		if kind == "launch" {
			allowed = nil
		}
		if kind == "opportunity" {
			allowed = []string{"title", "description", "organization", "type", "url", "country", "deadline", "status", "categoryId", "photo", "photoRights", "needsRightsReview", "languages", "forWhom"}
		}
		for _, k := range allowed {
			if v, ok := d[k]; ok {
				out[k] = v
			}
		}
		if kind == "opportunity" {
			if e = validateOpportunity(out); e != nil {
				return e
			}
		}
		out["updatedAt"] = now()
		if e = t.Put(r.Context(), kind, out); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, kind+".save", str(out["id"]), nil)
	})
	if e != nil {
		return e
	}
	respond(w, 200, out)
	return nil
}
func (s *Server) users(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if e = requireRole(u, "admin"); e != nil {
		return e
	}
	all, e := s.Store.List(r.Context(), "user")
	if e != nil {
		return e
	}
	out := []Doc{}
	for _, v := range all {
		out = append(out, safeUser(v))
	}
	respond(w, 200, Doc{"items": out})
	return nil
}
func (s *Server) editUser(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if e = requireRole(u, "admin"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	for _, role := range array(d["roles"]) {
		if !has([]string{"reader", "author", "editor", "chief_editor", "admin"}, str(role)) {
			return errStatus(400, "Unknown role")
		}
	}
	var out Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		out, e = t.Get(r.Context(), "user", r.PathValue("id"))
		if e != nil {
			return e
		}
		if v, ok := d["roles"]; ok {
			out["roles"] = v
		}
		if v, ok := d["verticals"]; ok {
			out["verticals"] = v
		}
		if e = t.Put(r.Context(), "user", out); e != nil {
			return e
		}
		if has(out["roles"], "author") || has(out["roles"], "editor") || has(out["roles"], "chief_editor") {
			if _, profileErr := t.Get(r.Context(), "author", str(out["id"])); profileErr != nil {
				if e = t.Put(r.Context(), "author", Doc{"id": out["id"], "name": out["name"], "bio": localized(""), "role": "author"}); e != nil {
					return e
				}
			}
		}
		return t.Audit(r.Context(), u, "user.roles", str(out["id"]), d)
	})
	if e != nil {
		return e
	}
	respond(w, 200, safeUser(out))
	return nil
}
func (s *Server) analytics(w http.ResponseWriter, r *http.Request) error {
	if _, e := s.require(r, true); e != nil {
		return e
	}
	d := Doc{"published": 0, "stories": 0, "subscribers": 0, "views": 0, "saves": 0, "opportunityClicks": 0, "statuses": Doc{}, "daily": []any{}, "aiCostUsd": 0}
	for field, kind := range map[string]string{"published": "publication", "stories": "story", "saves": "save"} {
		var n int
		if e := s.Store.Pool.QueryRow(r.Context(), "SELECT count(*) FROM creata_records WHERE kind=$1", kind).Scan(&n); e != nil {
			return e
		}
		d[field] = n
	}
	var subscribers int
	if e := s.Store.Pool.QueryRow(r.Context(), "SELECT count(*) FROM creata_records WHERE kind='user' AND data->>'telegramId' IS NOT NULL AND COALESCE((data->>'paused')::boolean,false)=false").Scan(&subscribers); e != nil {
		return e
	}
	d["subscribers"] = subscribers
	for field, kind := range map[string]string{"views": "view", "opportunityClicks": "opportunity_click"} {
		var n int
		if e := s.Store.Pool.QueryRow(r.Context(), "SELECT count(*) FROM creata_events WHERE kind=$1", kind).Scan(&n); e != nil {
			return e
		}
		d[field] = n
	}
	articles, e := s.Store.List(r.Context(), "article")
	if e != nil {
		return e
	}
	statuses := Doc{}
	for _, a := range articles {
		k := str(a["status"])
		statuses[k] = integer(statuses[k]) + 1
	}
	d["statuses"] = statuses
	audits, e := s.Store.List(r.Context(), "audit")
	if e != nil {
		return e
	}
	d["stageMetrics"] = editorialStageMetrics(articles, audits, time.Now())
	var cost float64
	if e = s.Store.Pool.QueryRow(r.Context(), "SELECT COALESCE(sum(cost),0) FROM creata_ai_reservations WHERE committed").Scan(&cost); e != nil {
		return e
	}
	d["aiCostUsd"] = cost
	rows, e := s.Store.Pool.Query(r.Context(), "SELECT to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD'),kind,count(*) FROM creata_events WHERE created_at>now()-interval '30 days' GROUP BY 1,2 ORDER BY 1")
	if e != nil {
		return e
	}
	defer rows.Close()
	daily := []Doc{}
	for rows.Next() {
		var day, kind string
		var n int
		if e = rows.Scan(&day, &kind, &n); e != nil {
			return e
		}
		daily = append(daily, Doc{"date": day, "type": kind, "count": n})
	}
	d["daily"] = daily
	respond(w, 200, d)
	return rows.Err()
}
func (s *Store) Seed(ctx context.Context, b []byte) error {
	var seed map[string][]Doc
	if e := json.Unmarshal(b, &seed); e != nil {
		return e
	}
	return s.Write(ctx, func(t *Tx) error {
		for _, record := range []struct{ field, kind string }{{"categories", "category"}, {"authors", "author"}, {"opportunities", "opportunity"}, {"articles", "article"}} {
			field, kind := record.field, record.kind
			for _, d := range seed[field] {
				if str(d["id"]) == "" {
					d["id"] = id()
				}
				if str(d["createdAt"]) == "" {
					d["createdAt"] = now()
				}
				if kind == "article" {
					a := newArticle(Doc{"id": d["authorId"]}, d)
					a["id"] = d["id"]
					a["status"] = d["status"]
					a["revision"] = 1
					if str(a["status"]) == "published" {
						a["approvedRevision"] = 1
						if e := publish(t, ctx, a, "Development reference seed"); e != nil {
							return e
						}
					}
					if e := t.saveRevision(ctx, a, Doc{"id": "seed"}, "Development reference seed"); e != nil {
						return e
					}
				} else if e := t.Put(ctx, kind, d); e != nil {
					return e
				}
			}
		}
		return nil
	})
}
