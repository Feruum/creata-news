package news

import (
	"encoding/json"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

func classificationTexts(story Doc) map[string]string {
	texts := map[string]string{}
	for _, raw := range array(story["evidence"]) {
		e := object(raw)
		if str(e["url"]) != "" && strings.TrimSpace(str(e["text"])) != "" {
			texts[str(e["url"])] = strings.Join(strings.Fields(str(e["text"])), " ")
		}
	}
	if str(story["url"]) != "" && strings.TrimSpace(str(story["text"])) != "" {
		if _, exists := texts[str(story["url"])]; !exists {
			texts[str(story["url"])] = strings.Join(strings.Fields(str(story["text"])), " ")
		}
	}
	return texts
}
func classificationArray(v any, limit int) ([]any, bool) {
	switch v.(type) {
	case []any, []string:
		a := array(v)
		return a, len(a) <= limit
	}
	return nil, false
}
func validateClassification(result, story Doc) error {
	switch result["score"].(type) {
	case float64, int, int64, json.Number:
	default:
		return errStatus(400, "Classification score must be numeric")
	}
	score := number(result["score"])
	if math.IsNaN(score) || math.IsInf(score, 0) || score < 0 || score > 100 {
		return errStatus(400, "Classification score must be 0–100")
	}
	for field, limit := range map[string]int{"categoryId": 100, "region": 200, "reason": 4000} {
		value, ok := result[field].(string)
		if !ok || utf8.RuneCountInString(value) > limit || (field != "region" && strings.TrimSpace(value) == "") {
			return errStatus(400, "Invalid classification "+field)
		}
	}
	tags, ok := classificationArray(result["tags"], 30)
	if !ok {
		return errStatus(400, "Classification tags must be an array")
	}
	for _, tag := range tags {
		value, ok := tag.(string)
		if !ok || strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > 100 {
			return errStatus(400, "Classification tags must be nonempty strings")
		}
	}
	texts := classificationTexts(story)
	for _, field := range []string{"facts", "entities"} {
		// Earlier collectors omitted these optional lists. Normalize omission only;
		// malformed or unsupported supplied entries are always rejected.
		if _, exists := result[field]; !exists {
			result[field] = []any{}
		}
		values, ok := classificationArray(result[field], 50)
		if !ok {
			return errStatus(400, "Classification "+field+" must contain at most 50 objects")
		}
		for _, raw := range values {
			value := object(raw)
			if len(value) == 0 {
				return errStatus(400, "Classification "+field+" entries must be objects")
			}
			urls, ok := classificationArray(value["sourceUrls"], 10)
			if !ok || len(urls) == 0 {
				return errStatus(400, "Classification provenance requires source URLs")
			}
			seen := map[string]bool{}
			for _, rawURL := range urls {
				url, ok := rawURL.(string)
				if !ok || url == "" || texts[url] == "" || seen[url] {
					return errStatus(400, "Classification cites unavailable or duplicate source evidence")
				}
				if _, e := canonicalURL(url); e != nil {
					return errStatus(400, "Invalid classification source URL")
				}
				seen[url] = true
			}
			if field == "facts" {
				claim, ok := value["claim"].(string)
				uncertainty, typed := value["uncertainty"].(string)
				if !ok || strings.TrimSpace(claim) == "" || utf8.RuneCountInString(claim) > 1000 || !typed || utf8.RuneCountInString(uncertainty) > 1000 {
					return errStatus(400, "A classification fact requires claim, uncertainty and source provenance")
				}
				for url := range seen {
					if !strings.Contains(texts[url], strings.Join(strings.Fields(claim), " ")) {
						return errStatus(400, "Classification claim must be a verbatim supplied source excerpt")
					}
				}
			} else {
				name, ok := value["name"].(string)
				if !ok || strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 200 || !has([]string{"person", "organization", "place", "product", "event", "other"}, str(value["type"])) {
					return errStatus(400, "Invalid classification named entity")
				}
				phrase := regexp.MustCompile(`(?i)(^|[^\pL\pN_])` + regexp.QuoteMeta(strings.Join(strings.Fields(name), " ")) + `($|[^\pL\pN_])`)
				for url := range seen {
					if !phrase.MatchString(texts[url]) {
						return errStatus(400, "Named entity does not occur in supplied source evidence")
					}
				}
			}
		}
	}
	return nil
}
