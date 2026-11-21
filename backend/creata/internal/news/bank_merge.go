package news

import (
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"
)

func storyWords(text string) map[string]bool {
	out := map[string]bool{}
	for _, word := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if len([]rune(word)) > 2 {
			out[word] = true
		}
	}
	return out
}
func nearStories(a, b Doc) (float64, []string) {
	if str(a["mergedInto"]) != "" || str(b["mergedInto"]) != "" || locale(str(a["locale"])) != locale(str(b["locale"])) {
		return 0, nil
	}
	at, bt := parseTime(a["publishedAt"]), parseTime(b["publishedAt"])
	if !at.IsZero() && !bt.IsZero() && at.Sub(bt).Abs() > 7*24*time.Hour {
		return 0, nil
	}
	aw, bw := storyWords(str(a["title"])), storyWords(str(b["title"]))
	common := 0
	for w := range aw {
		if bw[w] {
			common++
		}
	}
	union := len(aw) + len(bw) - common
	if union == 0 {
		return 0, nil
	}
	similarity := float64(common) / float64(union)
	if common < 4 || similarity < 0.65 {
		return 0, nil
	}
	reasons := []string{"Strong overlap in headline terms; editor must compare source claims"}
	if !at.IsZero() && !bt.IsZero() {
		reasons = append(reasons, "Publication dates are within seven days")
	}
	if str(a["articleId"]) != "" || str(b["articleId"]) != "" {
		reasons = append(reasons, "Existing article links will be retained")
	}
	return similarity, reasons
}
func (s *Server) bankList(w http.ResponseWriter, r *http.Request) error {
	if _, e := s.require(r, true); e != nil {
		return e
	}
	all, e := s.Store.List(r.Context(), "story")
	if e != nil {
		return e
	}
	out := []Doc{}
	for _, d := range all {
		if str(d["mergedInto"]) == "" {
			out = append(out, d)
		}
	}
	respond(w, 200, Doc{"items": out, "total": len(out)})
	return nil
}
func (s *Server) bankSimilar(w http.ResponseWriter, r *http.Request) error {
	if _, e := s.require(r, true); e != nil {
		return e
	}
	stories, e := s.Store.List(r.Context(), "story")
	if e != nil {
		return e
	}
	sort.Slice(stories, func(i, j int) bool { return str(stories[i]["id"]) < str(stories[j]["id"]) })
	if len(stories) > 1000 {
		stories = stories[:1000]
	}
	items := []Doc{}
	for i, a := range stories {
		for _, b := range stories[i+1:] {
			score, reasons := nearStories(a, b)
			if score > 0 {
				items = append(items, Doc{"leftId": a["id"], "rightId": b["id"], "leftTitle": a["title"], "rightTitle": b["title"], "leftUpdatedAt": a["updatedAt"], "rightUpdatedAt": b["updatedAt"], "score": score, "reasons": reasons})
			}
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if number(items[i]["score"]) == number(items[j]["score"]) {
			return str(items[i]["leftId"])+str(items[i]["rightId"]) < str(items[j]["leftId"])+str(items[j]["rightId"])
		}
		return number(items[i]["score"]) > number(items[j]["score"])
	})
	if len(items) > 100 {
		items = items[:100]
	}
	respond(w, 200, Doc{"items": items})
	return nil
}
func uniqueDocs(left, right any, key string) []any {
	out := []any{}
	seen := map[string]bool{}
	for _, values := range []any{left, right} {
		for _, raw := range array(values) {
			d := object(raw)
			v := str(d[key])
			if !seen[v] {
				out = append(out, raw)
				seen[v] = true
			}
		}
	}
	return out
}
func linkedArticles(stories ...Doc) []any {
	out := []any{}
	for _, s := range stories {
		for _, raw := range append(array(s["linkedArticleIds"]), s["articleId"], s["graphicId"]) {
			v := str(raw)
			if v != "" && !has(out, v) {
				out = append(out, v)
			}
		}
	}
	return out
}
func mergeAllowed(u Doc, stories ...Doc) error {
	if has(u["roles"], "admin") || has(u["roles"], "chief_editor") {
		return nil
	}
	for _, s := range stories {
		if !has(u["roles"], "editor") || !has(u["verticals"], str(s["categoryId"])) {
			return errStatus(403, "Story merge requires an editor for both verticals")
		}
	}
	return nil
}
func (s *Server) bankMerge(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	if str(d["otherId"]) == r.PathValue("id") {
		return errStatus(400, "Choose two different stories")
	}
	var result Doc
	e = s.Store.Write(r.Context(), func(tx *Tx) error {
		left, e := tx.Get(r.Context(), "story", r.PathValue("id"))
		if e != nil {
			return e
		}
		right, e := tx.Get(r.Context(), "story", str(d["otherId"]))
		if e != nil {
			return e
		}
		if e = mergeAllowed(u, left, right); e != nil {
			return e
		}
		if _, ok := d["expectedUpdatedAt"]; !ok {
			return errStatus(400, "Story versions are required")
		}
		if _, ok := d["otherExpectedUpdatedAt"]; !ok {
			return errStatus(400, "Story versions are required")
		}
		if str(left["updatedAt"]) != str(d["expectedUpdatedAt"]) || str(right["updatedAt"]) != str(d["otherExpectedUpdatedAt"]) || str(left["mergedInto"]) != "" || str(right["mergedInto"]) != "" {
			return errStatus(409, "Stories changed; refresh before merging")
		}
		result = Doc{"id": id(), "leftId": left["id"], "rightId": right["id"], "beforeLeft": clone(left), "beforeRight": clone(right), "createdAt": now(), "actorId": u["id"], "status": "merged"}
		rows, e := tx.Query(r.Context(), "SELECT fingerprint FROM creata_fingerprints WHERE story_id=$1", right["id"])
		if e != nil {
			return e
		}
		fps := []string{}
		for rows.Next() {
			var fp string
			if e = rows.Scan(&fp); e != nil {
				rows.Close()
				return e
			}
			fps = append(fps, fp)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		result["fingerprints"] = fps
		left["sources"] = uniqueDocs(left["sources"], right["sources"], "url")
		left["evidence"] = uniqueDocs(left["evidence"], right["evidence"], "url")
		left["linkedArticleIds"] = linkedArticles(left, right)
		ids := append(array(left["mergedStoryIds"]), str(right["id"]))
		for _, sid := range array(right["mergedStoryIds"]) {
			if !has(ids, str(sid)) {
				ids = append(ids, sid)
			}
		}
		left["mergedStoryIds"] = ids
		left["updatedAt"] = now()
		right["mergedInto"] = left["id"]
		right["status"] = "merged"
		right["updatedAt"] = now()
		result["afterLeftHash"] = hashJSON(left)
		result["afterRightHash"] = hashJSON(right)
		if e = tx.Put(r.Context(), "story", left); e != nil {
			return e
		}
		if e = tx.Put(r.Context(), "story", right); e != nil {
			return e
		}
		if _, e = tx.Exec(r.Context(), "UPDATE creata_fingerprints SET story_id=$1 WHERE story_id=$2", left["id"], right["id"]); e != nil {
			return e
		}
		if e = tx.Put(r.Context(), "story_merge", result); e != nil {
			return e
		}
		return tx.Audit(r.Context(), u, "story.merge", str(left["id"]), Doc{"mergeId": result["id"], "otherId": right["id"]})
	})
	if e != nil {
		return e
	}
	respond(w, 200, mergePublic(result))
	return nil
}
func mergePublic(d Doc) Doc {
	return Doc{"id": d["id"], "leftId": d["leftId"], "rightId": d["rightId"], "status": d["status"], "createdAt": d["createdAt"], "undoneAt": d["undoneAt"]}
}
func (s *Server) bankMerges(w http.ResponseWriter, r *http.Request) error {
	if _, e := s.require(r, true); e != nil {
		return e
	}
	all, e := s.Store.List(r.Context(), "story_merge")
	if e != nil {
		return e
	}
	out := []Doc{}
	for _, m := range all {
		out = append(out, mergePublic(m))
	}
	respond(w, 200, Doc{"items": out})
	return nil
}
func (s *Server) bankUndo(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	var result Doc
	e = s.Store.Write(r.Context(), func(tx *Tx) error {
		result, e = tx.Get(r.Context(), "story_merge", r.PathValue("id"))
		if e != nil {
			return e
		}
		if str(result["status"]) != "merged" {
			return errStatus(409, "Merge was already undone")
		}
		left, e := tx.Get(r.Context(), "story", str(result["leftId"]))
		if e != nil {
			return e
		}
		right, e := tx.Get(r.Context(), "story", str(result["rightId"]))
		if e != nil {
			return e
		}
		if e = mergeAllowed(u, left, right); e != nil {
			return e
		}
		if hashJSON(left) != str(result["afterLeftHash"]) || hashJSON(right) != str(result["afterRightHash"]) {
			return errStatus(409, "Stories changed since merge; split requires editorial review to preserve new evidence")
		}
		for _, fp := range array(result["fingerprints"]) {
			if _, e = tx.Exec(r.Context(), "UPDATE creata_fingerprints SET story_id=$1 WHERE fingerprint=$2 AND story_id=$3", right["id"], fp, left["id"]); e != nil {
				return e
			}
		}
		if e = tx.Put(r.Context(), "story", object(result["beforeLeft"])); e != nil {
			return e
		}
		if e = tx.Put(r.Context(), "story", object(result["beforeRight"])); e != nil {
			return e
		}
		result["status"] = "undone"
		result["undoneAt"] = now()
		if e = tx.Put(r.Context(), "story_merge", result); e != nil {
			return e
		}
		return tx.Audit(r.Context(), u, "story.split", str(left["id"]), Doc{"mergeId": result["id"]})
	})
	if e != nil {
		return e
	}
	respond(w, 200, mergePublic(result))
	return nil
}
func (s *Server) bankEvidence(w http.ResponseWriter, r *http.Request) error {
	if _, e := s.require(r, true); e != nil {
		return e
	}
	story, e := s.Store.Get(r.Context(), "story", r.PathValue("id"))
	if e != nil {
		return e
	}
	raw, e := s.Store.List(r.Context(), "raw_item")
	if e != nil {
		return e
	}
	out := []Doc{}
	for _, item := range raw {
		if str(item["storyId"]) == str(story["id"]) || has(story["mergedStoryIds"], str(item["storyId"])) {
			out = append(out, item)
		}
	}
	respond(w, 200, Doc{"story": story, "rawItems": out, "linkedArticleIds": linkedArticles(story), "version": hashJSON(story)})
	return nil
}
