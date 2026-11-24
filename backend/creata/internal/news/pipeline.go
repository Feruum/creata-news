package news

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func hashJSON(v any) string { b, _ := json.Marshal(v); return hash(string(b)) }

var researchNumber = regexp.MustCompile(`\d+(?:[.,]\d+)*`)

func validateResearchBrief(brief Doc) error {
	evidence := map[string]string{}
	for _, raw := range array(brief["retrievedEvidence"]) {
		e := object(raw)
		if _, err := canonicalURL(str(e["url"])); err != nil || strings.TrimSpace(str(e["text"])) == "" {
			return errStatus(400, "Research requires retrieved source text")
		}
		evidence[str(e["url"])] = str(e["text"])
	}
	if len(evidence) == 0 {
		return errStatus(400, "Research requires retrieved evidence")
	}
	if _, ok := brief["facts"].([]any); !ok {
		return errStatus(400, "Research facts must be an array")
	}
	for _, raw := range array(brief["facts"]) {
		fact := object(raw)
		if strings.TrimSpace(str(fact["claim"])) == "" || len(array(fact["sourceUrls"])) == 0 {
			return errStatus(400, "Each research claim needs source provenance")
		}
		for _, u := range array(fact["sourceUrls"]) {
			if _, ok := evidence[str(u)]; !ok {
				return errStatus(400, "Research claim cites unavailable source")
			}
		}
		urls := map[string]bool{}
		for _, u := range array(fact["sourceUrls"]) {
			urls[str(u)] = true
		}
		if len(urls) < 2 && !has(brief["singleSourceClaims"], str(fact["claim"])) {
			return errStatus(400, "Single-source research claims must be flagged")
		}
		supported := map[string]bool{}
		for _, field := range []string{"numbers", "dates"} {
			for _, v := range array(fact[field]) {
				ref := object(v)
				text := evidence[str(ref["sourceUrl"])]
				quote := str(ref["quote"])
				value := str(ref["value"])
				if !has(fact["sourceUrls"], str(ref["sourceUrl"])) || quote == "" || value == "" || !strings.Contains(text, quote) || !strings.Contains(quote, value) {
					return errStatus(400, "Research numbers and dates require an exact source quote")
				}
				for _, n := range researchNumber.FindAllString(value, -1) {
					supported[n] = true
				}
			}
		}
		for _, n := range researchNumber.FindAllString(str(fact["claim"]), -1) {
			if !supported[n] {
				return errStatus(400, "Every research number and date requires source evidence")
			}
		}
	}
	for _, field := range []string{"conflicts", "singleSourceClaims", "unavailableSources"} {
		if brief[field] == nil {
			return errStatus(400, "Research requires "+field)
		}
		if _, ok := brief[field].([]any); !ok {
			if _, ok = brief[field].([]string); !ok {
				return errStatus(400, "Research warnings must be arrays")
			}
		}
		for _, v := range array(brief[field]) {
			if _, ok := v.(string); !ok {
				return errStatus(400, "Research warnings must be text")
			}
		}
	}
	return nil
}

func addExaResearchSources(article, brief Doc) {
	cited := map[string]bool{}
	for _, fact := range array(brief["facts"]) {
		for _, source := range array(object(fact)["sourceUrls"]) {
			cited[str(source)] = true
		}
	}
	sources := array(article["sources"])
	seen := map[string]bool{}
	for _, source := range sources {
		if url, err := canonicalURL(str(object(source)["url"])); err == nil {
			seen[url] = true
		}
	}
	added := 0
	for _, value := range array(brief["retrievedEvidence"]) {
		if added == 2 || len(sources) >= 50 {
			break
		}
		evidence := object(value)
		rawURL := str(evidence["url"])
		if str(evidence["discoveredBy"]) != "exa" || !cited[rawURL] || !validArticleSourceURL(rawURL) {
			continue
		}
		url, err := canonicalURL(rawURL)
		title := strings.TrimSpace(str(evidence["title"]))
		if err != nil || seen[url] || title == "" {
			continue
		}
		source := Doc{"url": url, "title": title}
		if date := str(evidence["publishedAt"]); date != "" {
			source["publishedAt"] = date
		}
		sources = append(sources, source)
		seen[url] = true
		added++
	}
	if added > 0 {
		article["sources"] = sources
	}
}

func checkpoint(j, d Doc) error {
	phase, status := str(d["phase"]), str(d["status"])
	if !has([]string{"research", "draft", "review", "classification"}, phase) || !has([]string{"running", "succeeded", "failed"}, status) || (phase == "classification") != (str(j["kind"]) == "classify") {
		return errStatus(400, "Invalid pipeline checkpoint")
	}
	checkpoints := object(j["checkpoints"])
	previous := object(checkpoints[phase])
	if str(previous["status"]) == "succeeded" {
		if status == "succeeded" && hashJSON(previous["output"]) == hashJSON(d["output"]) {
			return nil
		}
		return errStatus(409, "Completed pipeline phases are immutable")
	}
	if phase != "research" && phase != "classification" && str(object(checkpoints["research"])["status"]) != "succeeded" {
		return errStatus(409, "Research phase must complete first")
	}
	if phase == "review" && str(object(checkpoints["draft"])["status"]) != "succeeded" {
		return errStatus(409, "Draft phase must complete first")
	}
	next := Doc{"status": status, "updatedAt": now()}
	if d["usage"] != nil {
		usage, ok := classificationArray(d["usage"], 10)
		if !ok {
			return errStatus(400, "Invalid accounting usage list")
		}
		for _, raw := range usage {
			entry := object(raw)
			if str(entry["reservationId"]) == "" || number(entry["costUsd"]) < 0 || integer(entry["inputTokens"]) < 0 || integer(entry["outputTokens"]) < 0 {
				return errStatus(400, "Invalid accounting usage")
			}
		}
		next["usage"] = usage
	}
	if status == "succeeded" {
		output := object(d["output"])
		if len(output) == 0 {
			return errStatus(400, "Checkpoint output is required")
		}
		if phase == "research" {
			if e := validateResearchBrief(output); e != nil {
				return e
			}
		}
		if phase == "classification" {
			if e := validateClassification(output, object(object(j["payload"])["story"])); e != nil {
				return e
			}
		}
		if phase == "draft" {
			for _, field := range []string{"title", "lead", "body", "summary", "actions"} {
				for _, lang := range []string{"ru", "kk", "en"} {
					if strings.TrimSpace(str(object(output[field])[lang])) == "" {
						return errStatus(400, "Draft checkpoint requires all translations")
					}
				}
			}
		}
		if phase == "review" {
			if output["reviewNotes"] == nil {
				return errStatus(400, "Review notes are required")
			}
			for _, v := range array(output["reviewNotes"]) {
				if _, ok := v.(string); !ok {
					return errStatus(400, "Review notes must be strings")
				}
			}
		}
		next["output"] = output
		next["completedAt"] = now()
	} else if status == "failed" {
		next["error"] = str(d["error"])
	}
	checkpoints[phase] = next
	j["checkpoints"] = checkpoints
	j["progress"] = Doc{"phase": phase, "status": status, "updatedAt": now()}
	return nil
}

// Opportunities are selected from persisted approved records, never invented by a model.
func eligibleOpportunities(t *Tx, ctx context.Context, article, story Doc) ([]Doc, error) {
	all, e := t.List(ctx, "opportunity")
	if e != nil {
		return nil, e
	}
	out := []Doc{}
	category := str(article["categoryId"])
	if category == "" {
		category = str(story["categoryId"])
	}
	language := locale(str(story["locale"]))
	for _, o := range all {
		if str(o["status"]) != "approved" || !parseTime(o["deadline"]).After(time.Now()) || str(o["categoryId"]) != category {
			continue
		}
		if o["languages"] != nil && !has(o["languages"], language) {
			continue
		}
		if e := validateOpportunity(o); e != nil {
			continue
		}
		out = append(out, o)
		if len(out) == 12 {
			break
		}
	}
	return out, nil
}

func attachJobOpportunities(t *Tx, ctx context.Context, j Doc) error {
	if !has([]string{"draft", "rework"}, str(j["kind"])) {
		return nil
	}
	p := object(j["payload"])
	o, e := eligibleOpportunities(t, ctx, object(p["article"]), object(p["story"]))
	if e != nil {
		return e
	}
	p["opportunities"] = o
	j["payload"] = p
	return nil
}

func enqueuePipelineJob(t *Tx, ctx context.Context, j Doc) error {
	if e := attachJobOpportunities(t, ctx, j); e != nil {
		return e
	}
	p := object(j["payload"])
	article := object(p["article"])
	if has([]string{"draft", "rework"}, str(j["kind"])) && str(article["id"]) != "" {
		current, e := t.Get(ctx, "article", str(article["id"]))
		if e != nil {
			return e
		}
		current["activeAIJobId"] = j["id"]
		article["activeAIJobId"] = j["id"]
		p["expectedAIJobId"] = j["id"]
		if e = t.Put(ctx, "article", current); e != nil {
			return e
		}
	}
	return t.Put(ctx, "job", j)
}

func validateJobOpportunities(t *Tx, ctx context.Context, p, result Doc) error {
	if result["opportunityIds"] != nil {
		if _, ok := result["opportunityIds"].([]any); !ok {
			if _, ok = result["opportunityIds"].([]string); !ok {
				return errStatus(400, "Opportunity identifiers must be an array")
			}
		}
	}
	allowed := map[string]bool{}
	for _, raw := range array(p["opportunities"]) {
		allowed[str(object(raw)["id"])] = true
	}
	seen := map[string]bool{}
	for _, raw := range array(result["opportunityIds"]) {
		oid := str(raw)
		if oid == "" || !allowed[oid] || seen[oid] {
			return errStatus(400, "AI selected an unknown or duplicate opportunity")
		}
		seen[oid] = true
		o, e := t.Get(ctx, "opportunity", oid)
		if e != nil || str(o["status"]) != "approved" || !parseTime(o["deadline"]).After(time.Now()) {
			return errStatus(400, "Selected opportunity is no longer available")
		}
	}
	return nil
}

func (s *Server) articleResearch(w http.ResponseWriter, r *http.Request) error {
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
	jobs, e := s.Store.List(r.Context(), "job")
	if e != nil {
		return e
	}
	out := []Doc{}
	for _, j := range jobs {
		if str(object(object(j["payload"])["article"])["id"]) == str(a["id"]) {
			c := clone(j)
			delete(c, "leaseToken")
			out = append(out, c)
		}
	}
	respond(w, 200, Doc{"items": out})
	return nil
}
