package news

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

func newJob(kind string, payload Doc) Doc {
	return Doc{"id": id(), "kind": kind, "status": "queued", "payload": payload, "attempts": 0, "createdAt": now(), "updatedAt": now()}
}
func (s *Server) jobRoutes() {
	s.handle("GET "+base+"/admin/sources", func(w http.ResponseWriter, r *http.Request) error {
		u, e := s.require(r, true)
		if e != nil {
			return e
		}
		if e = requireRole(u, "admin"); e != nil {
			return e
		}
		return s.listResponse(w, r, "source")
	})
	s.handle("POST "+base+"/admin/sources", s.sourceEdit)
	s.handle("PATCH "+base+"/admin/sources/{id}", s.sourceEdit)
	s.handle("POST "+base+"/admin/sources/{id}/{action}", s.sourceAction)
	for _, p := range []string{"jobs", "runs"} {
		s.handle("GET "+base+"/admin/"+p, s.jobsList)
	}
	s.handle("POST "+base+"/admin/jobs/{id}/{action}", s.jobAction)
	s.handle("GET "+base+"/admin/settings", s.settingsGet)
	s.handle("PATCH "+base+"/admin/settings", s.settingsPatch)
	s.handle("POST "+base+"/internal/jobs/claim", s.claimJob)
	s.handle("POST "+base+"/internal/jobs/{id}/{action}", s.finishJob)
	s.handle("GET "+base+"/internal/ai-config", s.aiConfig)
	s.handle("POST "+base+"/internal/ai/reserve", s.reserveAI)
	s.handle("POST "+base+"/internal/ai/commit", s.commitAI)
}
func (s *Server) sourceEdit(w http.ResponseWriter, r *http.Request) error {
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
	var out Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		if r.Method == "PATCH" {
			out, e = t.Get(r.Context(), "source", r.PathValue("id"))
			if e != nil {
				return e
			}
		} else {
			out = Doc{"id": id(), "type": "rss", "locale": "ru", "topics": []any{}, "intervalMinutes": 60, "active": false, "createdAt": now()}
		}
		for _, k := range []string{"name", "type", "url", "locale", "topics", "intervalMinutes", "active"} {
			if v, ok := d[k]; ok {
				out[k] = v
			}
		}
		if !has([]string{"rss", "website", "telegram"}, str(out["type"])) || !has([]string{"ru", "kk", "en"}, str(out["locale"])) {
			return errStatus(400, "Invalid source type or language")
		}
		normalized, err := canonicalURL(str(out["url"]))
		if err != nil {
			return errStatus(400, "Invalid source URL")
		}
		parsed, _ := url.Parse(normalized)
		if !s.Config.AllowPrivateSources && isLoopback(parsed.Hostname()) {
			return errStatus(400, "Private source targets are forbidden")
		}
		if str(out["type"]) == "telegram" && parsed.Hostname() != "t.me" && parsed.Hostname() != "telegram.me" {
			return errStatus(400, "Telegram sources must use public t.me channel URLs")
		}
		if integer(out["intervalMinutes"]) < 15 || integer(out["intervalMinutes"]) > 1440 {
			return errStatus(400, "Source interval must be 15–1440 minutes")
		}
		out["url"] = normalized
		out["nextRunAt"] = now()
		out["updatedAt"] = now()
		if e = t.Put(r.Context(), "source", out); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "source.save", str(out["id"]), nil)
	})
	if e != nil {
		return e
	}
	respond(w, 200, out)
	return nil
}
func (s *Server) sourceAction(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if e = requireRole(u, "admin"); e != nil {
		return e
	}
	action := r.PathValue("action")
	if action != "check" && action != "run" {
		return errStatus(400, "Unknown source action")
	}
	var job Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		src, e := t.Get(r.Context(), "source", r.PathValue("id"))
		if e != nil {
			return e
		}
		kind := "collect"
		if action == "check" {
			kind = "check"
		}
		job, e = enqueueSource(t, r.Context(), src, kind)
		if e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "source."+action, str(src["id"]), nil)
	})
	if e != nil {
		return e
	}
	respond(w, 202, job)
	return nil
}
func enqueueSource(t *Tx, ctx context.Context, src Doc, kind string) (Doc, error) {
	all, e := t.List(ctx, "job")
	if e != nil {
		return nil, e
	}
	for _, j := range all {
		if str(j["kind"]) == kind && (str(j["status"]) == "queued" || str(j["status"]) == "running") && str(object(object(j["payload"])["source"])["id"]) == str(src["id"]) {
			return j, nil
		}
	}
	j := newJob(kind, Doc{"source": src})
	return j, t.Put(ctx, "job", j)
}
func (s *Server) jobsList(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if e = requireRole(u, "admin"); e != nil {
		return e
	}
	items, e := s.Store.List(r.Context(), "job")
	if e != nil {
		return e
	}
	out := []Doc{}
	for _, j := range items {
		if strings.HasSuffix(r.URL.Path, "/runs") && str(j["kind"]) != "collect" && str(j["kind"]) != "check" {
			continue
		}
		c := clone(j)
		delete(c, "leaseToken")
		out = append(out, c)
	}
	respond(w, 200, Doc{"items": out})
	return nil
}
func (s *Server) jobAction(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if e = requireRole(u, "admin"); e != nil {
		return e
	}
	// Consume optional JSON before writing checkpoint-rich responses. An unread
	// request body with Connection: close can reset a Windows TCP connection.
	if r.ContentLength != 0 {
		if _, e = decode(r); e != nil {
			return e
		}
	}
	var j Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		j, e = t.Get(r.Context(), "job", r.PathValue("id"))
		if e != nil {
			return e
		}
		switch r.PathValue("action") {
		case "retry":
			if !has([]string{"failed", "not_configured", "cancelled"}, str(j["status"])) {
				return errStatus(409, "Only stopped jobs can be retried")
			}
			j["status"] = "queued"
			j["attempts"] = 0
			delete(j, "error")
			delete(j, "leaseToken")
			delete(j, "retryAt")
		case "cancel":
			if str(j["status"]) != "queued" {
				return errStatus(409, "Only queued jobs can be cancelled")
			}
			j["status"] = "cancelled"
		default:
			return errStatus(400, "Unknown job action")
		}
		j["updatedAt"] = now()
		return t.Put(r.Context(), "job", j)
	})
	if e != nil {
		return e
	}
	respond(w, 200, j)
	return nil
}
func (s *Server) claimJob(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "worker"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	if str(d["workerId"]) == "" {
		return errStatus(400, "workerId is required")
	}
	var found Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		rows, e := t.Query(r.Context(), "SELECT data FROM creata_records WHERE kind='job' AND (data->>'status'='queued' OR (data->>'status'='running' AND (data->>'leaseUntil')::timestamptz<now())) ORDER BY created_at FOR UPDATE SKIP LOCKED")
		if e != nil {
			return e
		}
		jobs := []Doc{}
		for rows.Next() {
			var b []byte
			if e = rows.Scan(&b); e != nil {
				rows.Close()
				return e
			}
			var j Doc
			if e = json.Unmarshal(b, &j); e != nil {
				rows.Close()
				return e
			}
			jobs = append(jobs, j)
		}
		rows.Close()
		for _, j := range jobs {
			if len(array(d["kinds"])) > 0 && !has(d["kinds"], str(j["kind"])) {
				continue
			}
			if parseTime(j["retryAt"]).After(time.Now()) {
				continue
			}
			if integer(j["attempts"]) >= 3 {
				j["status"] = "failed"
				j["error"] = "Retry limit reached after abandoned job"
				if e = t.Put(r.Context(), "job", j); e != nil {
					return e
				}
				continue
			}
			j["status"] = "running"
			j["workerId"] = d["workerId"]
			j["leaseToken"] = id() + id()
			j["leaseUntil"] = time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339Nano)
			j["attempts"] = integer(j["attempts"]) + 1
			j["updatedAt"] = now()
			if object(j["payload"])["opportunities"] == nil {
				if e = attachJobOpportunities(t, r.Context(), j); e != nil {
					return e
				}
			}
			found = j
			return t.Put(r.Context(), "job", j)
		}
		return nil
	})
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"job": found})
	return nil
}
func (s *Server) finishJob(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "worker"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	var j Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		j, e = t.Get(r.Context(), "job", r.PathValue("id"))
		if e != nil {
			return e
		}
		if str(d["leaseToken"]) == "" || str(d["leaseToken"]) != str(j["leaseToken"]) {
			return errStatus(409, "Job lease is no longer owned")
		}
		if str(j["status"]) == "succeeded" && r.PathValue("action") == "complete" {
			return nil
		}
		if str(j["status"]) != "running" || !parseTime(j["leaseUntil"]).After(time.Now()) {
			return errStatus(409, "Job lease expired")
		}
		switch r.PathValue("action") {
		case "checkpoint":
			if !has([]string{"draft", "rework", "classify"}, str(j["kind"])) {
				return errStatus(400, "Only AI jobs have checkpoints")
			}
			if str(d["phase"]) == "classification" && str(d["status"]) == "succeeded" {
				if _, e = t.Get(r.Context(), "category", str(object(d["output"])["categoryId"])); e != nil {
					return errStatus(400, "Unknown classification category")
				}
			}
			if e = checkpoint(j, d); e != nil {
				return e
			}
		case "heartbeat":
			j["leaseUntil"] = time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339Nano)
		case "fail":
			j["error"] = str(d["error"])
			j["status"] = "failed"
			if boolean(d["notConfigured"]) {
				j["status"] = "not_configured"
			} else if boolean(d["retryable"]) && integer(j["attempts"]) < 3 {
				j["status"] = "queued"
				j["retryAt"] = time.Now().Add(time.Duration(integer(j["attempts"])) * time.Minute).UTC().Format(time.RFC3339Nano)
			}
			src := object(object(j["payload"])["source"])
			if str(src["id"]) != "" {
				if current, err := t.Get(r.Context(), "source", str(src["id"])); err == nil {
					current["lastError"] = j["error"]
					if e = t.Put(r.Context(), "source", current); e != nil {
						return e
					}
				}
			}
		case "complete":
			if e = s.applyJobResult(t, r.Context(), j, object(d["result"])); e != nil {
				return e
			}
			j["status"] = "succeeded"
			j["result"] = d["result"]
			delete(j, "error")
		default:
			return errStatus(400, "Unknown worker action")
		}
		j["updatedAt"] = now()
		return t.Put(r.Context(), "job", j)
	})
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"ok": true, "job": j})
	return nil
}
func (s *Server) applyJobResult(t *Tx, ctx context.Context, j, result Doc) error {
	payload := object(j["payload"])
	switch str(j["kind"]) {
	case "collect", "check":
		src := object(payload["source"])
		if len(array(result["items"])) > 500 {
			return errStatus(400, "Source returned too many items")
		}
		for _, raw := range array(result["items"]) {
			item := object(raw)
			if e := s.ingest(t, ctx, src, item); e != nil {
				var inputError *apiError
				if errors.As(e, &inputError) && inputError.status == 400 {
					result["warnings"] = append(array(result["warnings"]), Doc{"url": item["url"], "error": inputError.message})
					continue
				}
				return e
			}
		}
		current, e := t.Get(ctx, "source", str(src["id"]))
		if e != nil {
			return e
		}
		current["lastRunAt"] = now()
		current["lastError"] = ""
		current["nextRunAt"] = time.Now().Add(time.Duration(integer(current["intervalMinutes"])) * time.Minute).UTC().Format(time.RFC3339)
		return t.Put(ctx, "source", current)
	case "classify":
		story, e := t.Get(ctx, "story", str(object(payload["story"])["id"]))
		if e != nil {
			return e
		}
		if str(story["mergedInto"]) != "" {
			return nil
		}
		if e = validateClassification(result, object(payload["story"])); e != nil {
			return e
		}
		if _, e = t.Get(ctx, "category", str(result["categoryId"])); e != nil {
			return errStatus(400, "Unknown classification category")
		}
		score := number(result["score"])
		if score < 0 || score > 100 {
			return errStatus(400, "Classification score must be 0–100")
		}
		for _, k := range []string{"categoryId", "region", "tags", "score", "reason", "entities", "facts"} {
			if v, ok := result[k]; ok {
				story[k] = v
			}
		}
		copyArticlePhotos(story, result)
		if str(story["articleId"]) == "" && str(story["graphicId"]) == "" {
			story["status"] = "classified"
		}
		story["updatedAt"] = now()
		settings, e := t.Get(ctx, "settings", "settings")
		if e != nil {
			return e
		}
		if score >= number(settings["scoreThreshold"]) && str(story["articleId"]) == "" {
			if _, err := t.Get(ctx, "author", "hermes"); err != nil {
				if e = t.Put(ctx, "author", Doc{"id": "hermes", "name": "Hermes editorial assistant", "bio": localized("Editorial drafts require human review"), "role": "author"}); e != nil {
					return e
				}
			}
			a := newArticle(Doc{"id": "hermes"}, Doc{"title": localized(str(story["title"])), "categoryId": story["categoryId"], "region": story["region"], "tags": story["tags"], "sources": story["sources"]})
			copyArticlePhotos(a, story)
			a["status"] = "writing"
			if e = t.saveRevision(ctx, a, Doc{"id": "hermes"}, "Automatic editorial candidate"); e != nil {
				return e
			}
			story["articleId"] = a["id"]
			story["status"] = "in_progress"
			if e = enqueuePipelineJob(t, ctx, newJob("draft", Doc{"story": story, "article": a, "expectedRevision": a["revision"]})); e != nil {
				return e
			}
		}
		return t.Put(ctx, "story", story)
	case "draft", "rework":
		a, e := t.Get(ctx, "article", str(object(payload["article"])["id"]))
		if e != nil {
			return e
		}
		if integer(payload["expectedRevision"]) != integer(a["revision"]) {
			return errStatus(409, "Article changed during AI processing; result was not applied")
		}
		if expected := str(payload["expectedAIJobId"]); expected != "" {
			if expected != str(a["activeAIJobId"]) {
				return errStatus(409, "A newer editorial decision superseded this AI job")
			}
		} else if !has([]string{"writing", "rework"}, str(a["status"])) || str(a["activeAIJobId"]) != "" {
			return errStatus(409, "This legacy AI job is no longer current")
		}
		if e = validateJobOpportunities(t, ctx, payload, result); e != nil {
			return e
		}
		if result["researchBrief"] != nil {
			if e = validateResearchBrief(object(result["researchBrief"])); e != nil {
				return e
			}
		}
		for _, f := range []string{"title", "lead", "body", "summary", "actions"} {
			for _, l := range []string{"ru", "kk", "en"} {
				if str(object(result[f])[l]) == "" && f != "actions" && f != "summary" {
					return errStatus(400, "AI output is missing "+f+" in "+l)
				}
			}
		}
		for _, f := range []string{"title", "lead", "body", "summary", "actions", "reviewNotes", "opportunityIds", "researchBrief"} {
			if v, ok := result[f]; ok {
				a[f] = v
			}
		}
		if _, exists := a["photos"]; !exists && str(j["kind"]) == "draft" && str(a["format"]) != "graphic" {
			copyArticlePhotos(a, result)
		}
		brief := object(result["researchBrief"])
		if saved := object(object(object(j["checkpoints"])["research"])["output"]); len(saved) > 0 {
			brief = saved
			a["researchBrief"] = saved
		}
		if len(brief) > 0 && str(a["format"]) != "graphic" {
			addExaResearchSources(a, brief)
		}
		if e = validateEditor(t, ctx, a); e != nil {
			return e
		}
		if str(a["format"]) == "graphic" {
			g := object(result["graphic"])
			g["verified"] = false
			if e := validateGraphic(g, false); e != nil {
				return e
			}
			a["graphic"] = g
		}
		revise(a)
		if e = validateArticle(a); e != nil {
			return e
		}
		return t.saveRevision(ctx, a, Doc{"id": "hermes"}, str(payload["note"]))
	}
	return errStatus(400, "Unsupported job kind")
}
func (s *Server) ingest(t *Tx, ctx context.Context, src, item Doc) error {
	u, e := canonicalURL(str(item["url"]))
	if e != nil || strings.TrimSpace(str(item["title"])) == "" || strings.TrimSpace(str(item["text"])) == "" {
		return errStatus(400, "Invalid collected material")
	}
	if len(str(item["text"])) > 1000000 {
		return errStatus(400, "Collected material exceeds size limit")
	}
	fingerprints := materialFingerprints(str(src["id"]), item, u)
	storyID := ""
	for _, fp := range fingerprints {
		var existing string
		if e = t.QueryRow(ctx, "SELECT story_id FROM creata_fingerprints WHERE fingerprint=$1", fp).Scan(&existing); e == nil {
			storyID = existing
			break
		}
	}
	sourceRef := Doc{"url": u, "title": item["title"], "publishedAt": item["publishedAt"]}
	if storyID != "" {
		story, e := t.Get(ctx, "story", storyID)
		if e != nil {
			return e
		}
		sources := array(story["sources"])
		found := false
		for _, raw := range sources {
			if str(object(raw)["url"]) == u {
				found = true
			}
		}
		if !found && len(sources) < 50 {
			story["sources"] = append(sources, sourceRef)
		}
		if e = retainEvidence(t, ctx, src, item, u, story); e != nil {
			return e
		}
		story["updatedAt"] = now()
		if e = t.Put(ctx, "story", story); e != nil {
			return e
		}
		for _, fp := range fingerprints {
			if _, e = t.Exec(ctx, "INSERT INTO creata_fingerprints(fingerprint,story_id) VALUES($1,$2) ON CONFLICT DO NOTHING", fp, storyID); e != nil {
				return e
			}
		}
		return nil
	}
	storyID = id()
	story := Doc{"id": storyID, "title": item["title"], "text": item["text"], "url": u, "sourceId": src["id"], "sourceType": src["type"], "locale": locale(str(item["locale"])), "categoryId": "", "region": "", "tags": []any{}, "score": 0, "reason": "", "status": "new", "sources": []any{sourceRef}, "createdAt": now(), "publishedAt": item["publishedAt"], "externalId": item["externalId"]}
	story["updatedAt"] = now()
	if e = retainEvidence(t, ctx, src, item, u, story); e != nil {
		return e
	}
	if e = t.Put(ctx, "story", story); e != nil {
		return e
	}
	for _, fp := range fingerprints {
		if _, e = t.Exec(ctx, "INSERT INTO creata_fingerprints(fingerprint,story_id) VALUES($1,$2)", fp, storyID); e != nil {
			return e
		}
	}
	return t.Put(ctx, "job", newJob("classify", Doc{"story": story}))
}
func retainEvidence(t *Tx, ctx context.Context, src, item Doc, u string, story Doc) error {
	raw := clone(item)
	raw["id"] = hash(str(src["id"]) + ":" + u + ":" + str(item["text"]))
	raw["storyId"] = story["id"]
	raw["sourceId"] = src["id"]
	raw["url"] = u
	raw["collectedAt"] = now()
	if _, e := t.Get(ctx, "raw_item", str(raw["id"])); e != nil {
		if e = t.Put(ctx, "raw_item", raw); e != nil {
			return e
		}
	}
	text := []rune(str(item["text"]))
	if len(text) > 100000 {
		text = text[:100000]
	}
	evidence := Doc{"url": u, "title": item["title"], "text": string(text), "publishedAt": item["publishedAt"], "author": item["author"], "sourceId": src["id"]}
	items := array(story["evidence"])
	replaced := false
	for i, v := range items {
		if str(object(v)["url"]) == u {
			items[i] = evidence
			replaced = true
			break
		}
	}
	if !replaced && len(items) < 10 {
		items = append(items, evidence)
	}
	story["evidence"] = items
	return nil
}
func (s *Server) settingsPublic(ctx context.Context) (Doc, error) {
	d, e := s.Store.Get(ctx, "settings", "settings")
	if e != nil {
		return nil, e
	}
	d = clone(d)
	d["morningDigestHour"] = morningDigestHour(d)
	key, e := s.unseal(str(d["_apiKeyEncrypted"]))
	if str(d["_apiKeyEncrypted"]) == "" {
		key = ""
		e = nil
	}
	if e != nil {
		return nil, e
	}
	delete(d, "_apiKeyEncrypted")
	ai := object(d["ai"])
	ai["keyConfigured"] = key != ""
	ai["exaKeyConfigured"] = s.exaKey(d) != ""
	delete(ai, "exaApiKey")
	delete(d, "_exaApiKeyEncrypted")
	var spent float64
	e = s.Store.Pool.QueryRow(ctx, "SELECT COALESCE(sum(CASE WHEN committed THEN cost ELSE reserved END),0) FROM creata_ai_reservations WHERE day=CURRENT_DATE").Scan(&spent)
	object(d["ai"])["spentTodayUsd"] = spent
	return d, e
}
func (s *Server) settingsGet(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if e = requireRole(u, "admin"); e != nil {
		return e
	}
	d, e := s.settingsPublic(r.Context())
	if e != nil {
		return e
	}
	respond(w, 200, d)
	return nil
}
func (s *Server) settingsPatch(w http.ResponseWriter, r *http.Request) error {
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
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		current, e := t.Get(r.Context(), "settings", "settings")
		if e != nil {
			return e
		}
		for _, k := range []string{"editorialInstructions", "scoreThreshold", "digestHour", "timezone"} {
			if v, ok := d[k]; ok {
				current[k] = v
			}
		}
		if number(current["scoreThreshold"]) < 0 || number(current["scoreThreshold"]) > 100 || integer(current["digestHour"]) < 0 || integer(current["digestHour"]) > 23 {
			return errStatus(400, "Invalid threshold or digest hour")
		}
		if value, present := d["morningDigestHour"]; present {
			hour, valid := value.(float64)
			if !valid || hour < 0 || hour > 23 || hour != float64(int(hour)) {
				return errStatus(400, "Invalid morning digest hour")
			}
			current["morningDigestHour"] = int(hour)
		}
		if _, e = time.LoadLocation(str(current["timezone"])); e != nil {
			return errStatus(400, "Unknown timezone")
		}
		ai := object(current["ai"])
		inputAI := object(d["ai"])
		for _, k := range []string{"endpoint", "model", "dailyBudgetUsd", "inputPricePerMillion", "outputPricePerMillion"} {
			if v, ok := inputAI[k]; ok {
				ai[k] = v
			}
		}
		if str(ai["endpoint"]) != "" {
			u, e := url.Parse(str(ai["endpoint"]))
			if e != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "https" && !(s.Config.Environment != "production" && u.Scheme == "http" && isLoopback(u.Hostname()))) {
				return errStatus(400, "AI endpoint must use HTTPS")
			}
		}
		if number(ai["dailyBudgetUsd"]) < 0 || number(ai["inputPricePerMillion"]) < 0 || number(ai["outputPricePerMillion"]) < 0 {
			return errStatus(400, "AI prices and budget must be nonnegative")
		}
		if key, ok := inputAI["apiKey"]; ok {
			sealed, e := s.seal(str(key))
			if e != nil {
				return e
			}
			current["_apiKeyEncrypted"] = sealed
		}
		exaValue, exaChanged := inputAI["exaApiKey"]
		if exaChanged {
			key, ok := exaValue.(string)
			if !ok || len(key) > 1024 || strings.IndexFunc(key, unicode.IsControl) >= 0 {
				return errStatus(400, "Invalid Exa API key")
			}
			key = strings.TrimSpace(key)
			if key == "" {
				delete(current, "_exaApiKeyEncrypted")
			} else {
				sealed, err := s.seal(key)
				if err != nil {
					return err
				}
				current["_exaApiKeyEncrypted"] = sealed
			}
		}
		current["ai"] = ai
		if e = t.Put(r.Context(), "settings", current); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "settings.save", "settings", Doc{"aiCredentialsChanged": inputAI["apiKey"] != nil, "exaCredentialsChanged": exaChanged})
	})
	if e != nil {
		return e
	}
	out, e := s.settingsPublic(r.Context())
	if e != nil {
		return e
	}
	respond(w, 200, out)
	return nil
}

func (s *Server) exaKey(settings Doc) string {
	if encrypted := str(settings["_exaApiKeyEncrypted"]); encrypted != "" {
		key, _ := s.unseal(encrypted)
		return key
	}
	return ""
}

func (s *Server) aiConfig(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "worker"); e != nil {
		return e
	}
	raw, e := s.Store.Get(r.Context(), "settings", "settings")
	if e != nil {
		return e
	}
	key := ""
	if str(raw["_apiKeyEncrypted"]) != "" {
		key, e = s.unseal(str(raw["_apiKeyEncrypted"]))
		if e != nil {
			return e
		}
	}
	ai := clone(object(raw["ai"]))
	ai["apiKey"] = key
	ai["exaApiKey"] = s.exaKey(raw)
	ai["editorialInstructions"] = raw["editorialInstructions"]
	cats, e := s.Store.List(r.Context(), "category")
	if e != nil {
		return e
	}
	ai["categories"] = cats
	respond(w, 200, ai)
	return nil
}
func (s *Server) reserveAI(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "worker"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	cost := number(d["estimatedCostUsd"])
	if str(d["requestId"]) == "" || cost <= 0 {
		return errStatus(400, "Request identifier and positive cost are required")
	}
	reservation := ""
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		if e := t.QueryRow(r.Context(), "SELECT id FROM creata_ai_reservations WHERE request_id=$1", d["requestId"]).Scan(&reservation); e == nil {
			return nil
		}
		settings, e := t.Get(r.Context(), "settings", "settings")
		if e != nil {
			return e
		}
		ai := object(settings["ai"])
		key, _ := s.unseal(str(settings["_apiKeyEncrypted"]))
		if str(ai["endpoint"]) == "" || str(ai["model"]) == "" || key == "" || number(ai["dailyBudgetUsd"]) <= 0 || number(ai["inputPricePerMillion"]) <= 0 || number(ai["outputPricePerMillion"]) <= 0 {
			return errStatus(503, "AI is not configured")
		}
		var spent float64
		if e = t.QueryRow(r.Context(), "SELECT COALESCE(sum(CASE WHEN committed THEN cost ELSE reserved END),0) FROM creata_ai_reservations WHERE day=CURRENT_DATE").Scan(&spent); e != nil {
			return e
		}
		if spent+cost > number(ai["dailyBudgetUsd"]) {
			return errStatus(429, "Daily AI budget exhausted")
		}
		reservation = id()
		_, e = t.Exec(r.Context(), "INSERT INTO creata_ai_reservations(id,request_id,reserved) VALUES($1,$2,$3)", reservation, d["requestId"], cost)
		return e
	})
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"reservationId": reservation})
	return nil
}
func (s *Server) commitAI(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "worker"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	if number(d["costUsd"]) < 0 {
		return errStatus(400, "Invalid usage cost")
	}
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		var committed bool
		if e := t.QueryRow(r.Context(), "SELECT committed FROM creata_ai_reservations WHERE id=$1 FOR UPDATE", d["reservationId"]).Scan(&committed); e != nil {
			return errStatus(404, "Reservation not found")
		}
		if committed {
			return nil
		}
		_, e := t.Exec(r.Context(), "UPDATE creata_ai_reservations SET cost=$2,input_tokens=$3,output_tokens=$4,committed=true WHERE id=$1", d["reservationId"], number(d["costUsd"]), integer(d["inputTokens"]), integer(d["outputTokens"]))
		return e
	})
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"ok": true})
	return nil
}
func (s *Server) Tick(ctx context.Context) error {
	return s.Store.Write(ctx, func(t *Tx) error {
		sources, e := t.List(ctx, "source")
		if e != nil {
			return e
		}
		for _, src := range sources {
			if boolean(src["active"]) && !parseTime(src["nextRunAt"]).After(time.Now()) {
				if _, e = enqueueSource(t, ctx, src, "collect"); e != nil {
					return e
				}
				src["nextRunAt"] = time.Now().Add(time.Duration(integer(src["intervalMinutes"])) * time.Minute).UTC().Format(time.RFC3339)
				if e = t.Put(ctx, "source", src); e != nil {
					return e
				}
			}
		}
		articles, e := t.List(ctx, "article")
		if e != nil {
			return e
		}
		for _, a := range articles {
			due := parseTime(a["scheduledAt"])
			if !due.IsZero() && !due.After(time.Now()) && integer(a["approvedRevision"]) == integer(a["revision"]) {
				e = validateEditor(t, ctx, a)
				if e == nil {
					e = publishable(a)
				}
				if e == nil {
					e = t.validateStoryPermalink(ctx, a)
					var conflict *apiError
					if e != nil && (!errors.As(e, &conflict) || conflict.status != 409) {
						return e
					}
				}
				if e != nil {
					from := str(a["status"])
					delete(a, "scheduledAt")
					if err := cancelScheduledPostiz(t, ctx, str(a["id"])); err != nil {
						return err
					}
					a["reviewNotes"] = append(array(a["reviewNotes"]), e.Error())
					a["status"] = "review"
					delete(a, "approvedRevision")
					if e = recordStage(t, ctx, a, from, Doc{"id": "scheduler"}); e != nil {
						return e
					}
				} else if e = publishEditorial(t, ctx, a, str(a["publicationNote"])); e != nil {
					return e
				}
				if e = t.Put(ctx, "article", a); e != nil {
					return e
				}
			}
		}
		_, e = t.Exec(ctx, "DELETE FROM creata_sessions WHERE expires_at<now()")
		if e != nil {
			return e
		}
		if _, e = t.Exec(ctx, "DELETE FROM creata_records WHERE kind IN ('oauth','telegram_link') AND (data->>'expiresAt')::timestamptz<now()"); e != nil {
			return e
		}
		return s.scheduleDigests(t, ctx)
	})
}
func (s *Server) RunScheduler(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		if e := s.Tick(ctx); e != nil {
			fmt.Printf("scheduler: %v\n", e)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
