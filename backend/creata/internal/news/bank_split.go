package news

import "net/http"

// Explicit split preserves changes collected after the merge; stale editors must reload.
func (s *Server) bankSplit(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	d, e := decode(r)
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
			return errStatus(409, "Merge was already split")
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
		if str(d["expectedLeftVersion"]) != hashJSON(left) || str(d["expectedRightVersion"]) != hashJSON(right) || str(left["mergedInto"]) != "" || str(right["mergedInto"]) != str(left["id"]) {
			return errStatus(409, "Stories changed; inspect both stories again before splitting")
		}
		beforeLeft, beforeRight := object(result["beforeLeft"]), object(result["beforeRight"])
		restoredRight := clone(beforeRight)
		for _, field := range []string{"sources", "evidence"} {
			originalLeft := map[string]bool{}
			originalRight := map[string]bool{}
			for _, v := range array(beforeLeft[field]) {
				originalLeft[str(object(v)["url"])] = true
			}
			for _, v := range array(beforeRight[field]) {
				originalRight[str(object(v)["url"])] = true
			}
			kept := []any{}
			donor := []any{}
			for _, v := range array(left[field]) {
				url := str(object(v)["url"])
				if originalRight[url] && !originalLeft[url] {
					donor = append(donor, v)
				} else {
					kept = append(kept, v)
				}
			}
			left[field] = uniqueDocs(kept, beforeLeft[field], "url")
			restoredRight[field] = uniqueDocs(donor, beforeRight[field], "url")
		}
		donorArticles := linkedArticles(beforeRight)
		originalArticles := linkedArticles(beforeLeft)
		keptArticles := []any{}
		for _, v := range linkedArticles(left) {
			if !has(donorArticles, str(v)) || has(originalArticles, str(v)) {
				keptArticles = append(keptArticles, v)
			}
		}
		left["linkedArticleIds"] = keptArticles
		removed := append(array(beforeRight["mergedStoryIds"]), right["id"])
		groups := []any{}
		for _, v := range array(left["mergedStoryIds"]) {
			if !has(removed, str(v)) {
				groups = append(groups, v)
			}
		}
		left["mergedStoryIds"] = groups
		left["updatedAt"] = now()
		restoredRight["updatedAt"] = now()
		if e = restoreSplitProvenance(tx, r.Context(), result, left, restoredRight); e != nil {
			return e
		}
		if e = tx.Put(r.Context(), "story", left); e != nil {
			return e
		}
		if e = tx.Put(r.Context(), "story", restoredRight); e != nil {
			return e
		}
		result["status"] = "split"
		result["undoneAt"] = now()
		if e = tx.Put(r.Context(), "story_merge", result); e != nil {
			return e
		}
		return tx.Audit(r.Context(), u, "story.split", str(left["id"]), Doc{"mergeId": result["id"], "preservedChanges": true})
	})
	if e != nil {
		return e
	}
	respond(w, 200, mergePublic(result))
	return nil
}
