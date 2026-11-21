package news

import (
	"math"
	"net/http"
	"strings"
)

func graphicText(value any, required bool) bool {
	d := object(value)
	for _, l := range []string{"ru", "kk", "en"} {
		v, ok := d[l].(string)
		if !ok || len(v) > 5000 || (required && strings.TrimSpace(v) == "") {
			return false
		}
	}
	return true
}

func materialFormat(a Doc) string {
	if str(a["format"]) == "" {
		return "article"
	}
	return str(a["format"])
}

// Data is structured, bounded and rendered by our own SVG renderer; arbitrary embeds are never accepted.
func validateGraphic(value any, publication bool) error {
	g := object(value)
	if !has([]string{"bar", "line", "area", "donut"}, str(g["type"])) {
		return errStatus(400, "Choose bar, line, area or donut")
	}
	labels, series := array(g["labels"]), array(g["series"])
	if len(labels) < 2 || len(labels) > 120 || len(series) < 1 || len(series) > 6 {
		return errStatus(400, "A graphic needs 2–120 rows and 1–6 series")
	}
	for _, label := range labels {
		if !graphicText(label, true) {
			return errStatus(400, "Each chart label requires ru, kk and en")
		}
	}
	if len(str(g["unit"])) > 60 || len(str(g["period"])) > 150 || len(str(g["sourceName"])) > 250 {
		return errStatus(400, "Graphic metadata is too long")
	}
	for _, key := range []string{"unit", "period", "sourceName"} {
		if _, ok := g[key].(string); !ok {
			return errStatus(400, "Graphic "+key+" must be a string")
		}
	}
	if _, e := canonicalURL(str(g["sourceUrl"])); e != nil {
		return errStatus(400, "A graphic requires a valid source URL")
	}
	if !graphicText(g["methodology"], publication) {
		return errStatus(400, "Describe the graphic methodology in ru, kk and en")
	}
	if _, ok := g["verified"].(bool); !ok {
		return errStatus(400, "Graphic verified must be boolean")
	}
	if publication && (!boolean(g["verified"]) || strings.TrimSpace(str(g["unit"])) == "" || strings.TrimSpace(str(g["period"])) == "" || strings.TrimSpace(str(g["sourceName"])) == "") {
		return errStatus(400, "Verify the data and add its source and period before approval")
	}
	for _, raw := range series {
		s := object(raw)
		values := array(s["values"])
		if !graphicText(s["name"], true) || len(values) != len(labels) {
			return errStatus(400, "Every series needs a translated name and a numeric value for each row")
		}
		total := 0.0
		for _, v := range values {
			n, ok := v.(float64)
			if !ok || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 1e15 {
				return errStatus(400, "Chart values must be finite numbers within ±1e15")
			}
			if str(g["type"]) == "donut" && n < 0 {
				return errStatus(400, "Donut values cannot be negative")
			}
			total += n
		}
		if str(g["type"]) == "donut" && (len(series) != 1 || total <= 0) {
			return errStatus(400, "A donut requires one positive-total series")
		}
	}
	return nil
}

func validateGraphicPublication(a Doc) error {
	if str(a["format"]) != "graphic" {
		return nil
	}
	if e := validateGraphic(a["graphic"], true); e != nil {
		return e
	}
	want, _ := canonicalURL(str(object(a["graphic"])["sourceUrl"]))
	for _, raw := range array(a["sources"]) {
		u, _ := canonicalURL(str(object(raw)["url"]))
		if u == want {
			return nil
		}
	}
	return errStatus(400, "The graphic data source must also appear in the article sources")
}

func (s *Server) graphicDraft(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	var job Doc
	e = s.Store.Write(r.Context(), func(tx *Tx) error {
		a, e := tx.Get(r.Context(), "article", r.PathValue("id"))
		if e != nil {
			return e
		}
		if !canEdit(u, a) {
			return errStatus(403, "Graphic access denied")
		}
		if str(a["format"]) != "graphic" {
			return errStatus(400, "This material is not a graphic")
		}
		if integer(d["expectedRevision"]) != integer(a["revision"]) {
			return errStatus(409, "Graphic changed; reload")
		}
		if len(array(a["sources"])) == 0 {
			return errStatus(400, "Add a readable data source first")
		}
		jobs, e := tx.List(r.Context(), "job")
		if e != nil {
			return e
		}
		for _, j := range jobs {
			p := object(j["payload"])
			if has([]string{"queued", "running"}, str(j["status"])) && str(object(p["article"])["id"]) == str(a["id"]) && integer(p["expectedRevision"]) == integer(a["revision"]) {
				job = clone(j)
				delete(job, "leaseToken")
				return nil
			}
		}
		job = newJob("draft", Doc{"article": clone(a), "expectedRevision": a["revision"], "note": str(d["note"])})
		if e := enqueuePipelineJob(tx, r.Context(), job); e != nil {
			return e
		}
		return tx.Audit(r.Context(), u, "graphic.draft", str(a["id"]), nil)
	})
	if e != nil {
		return e
	}
	respond(w, 202, job)
	return nil
}

func (s *Server) bankGraphic(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	var a Doc
	e = s.Store.Write(r.Context(), func(tx *Tx) error {
		story, e := tx.Get(r.Context(), "story", r.PathValue("id"))
		if e != nil {
			return e
		}
		if str(story["mergedInto"]) != "" {
			return errStatus(409, "Story was merged; open the surviving story")
		}
		if aid := str(story["graphicId"]); aid != "" {
			a, e = tx.Get(r.Context(), "article", aid)
			if e != nil {
				return e
			}
			if !canEdit(u, a) {
				return errStatus(403, "Graphic access denied")
			}
			return nil
		}
		a = newArticle(u, Doc{"format": "graphic", "title": localized(str(story["title"])), "categoryId": story["categoryId"], "sources": story["sources"], "region": story["region"], "tags": story["tags"]})
		if str(a["categoryId"]) == "" {
			a["categoryId"] = "ai"
		}
		if !canEdit(u, a) {
			return errStatus(403, "Vertical access denied")
		}
		a["status"] = "writing"
		if e := validateEditor(tx, r.Context(), a); e != nil {
			return e
		}
		if e := tx.saveRevision(r.Context(), a, u, "Graphic created from source story"); e != nil {
			return e
		}
		story["graphicId"] = a["id"]
		if e := tx.Put(r.Context(), "story", story); e != nil {
			return e
		}
		return enqueuePipelineJob(tx, r.Context(), newJob("draft", Doc{"article": a, "story": story, "expectedRevision": a["revision"]}))
	})
	if e != nil {
		return e
	}
	respond(w, 201, a)
	return nil
}
