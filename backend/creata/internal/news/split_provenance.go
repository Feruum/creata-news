package news

import (
	"context"
	"encoding/json"
	"strings"
)

func materialFingerprints(sourceID string, item Doc, url string) []string {
	out := []string{hash("url:" + url), hash("text:" + strings.Join(strings.Fields(str(item["text"])), " "))}
	if str(item["externalId"]) != "" {
		out = append(out, hash("external:"+sourceID+":"+str(item["externalId"])))
	}
	return out
}
func storySourceURLs(story Doc) map[string]bool {
	out := map[string]bool{}
	if str(story["url"]) != "" {
		out[str(story["url"])] = true
	}
	for _, field := range []string{"sources", "evidence"} {
		for _, raw := range array(story[field]) {
			if url := str(object(raw)["url"]); url != "" {
				out[url] = true
			}
		}
	}
	return out
}

// Source content is immutable; split only restores its story ownership metadata.
// A shared text fingerprint stays with the survivor when its retained raw supports it.
func restoreSplitProvenance(tx *Tx, ctx context.Context, merge, left, right Doc) error {
	beforeLeft, beforeRight := object(merge["beforeLeft"]), object(merge["beforeRight"])
	leftURLs, rightURLs := storySourceURLs(beforeLeft), storySourceURLs(beforeRight)
	exclusive := map[string]bool{}
	for url := range rightURLs {
		if !leftURLs[url] {
			exclusive[url] = true
		}
	}
	groups := []string{str(left["id"])}
	for _, raw := range array(left["mergedStoryIds"]) {
		groups = append(groups, str(raw))
	}
	rows, e := tx.Query(ctx, "SELECT data FROM creata_records WHERE kind='raw_item' AND data->>'storyId'=ANY($1::text[])", groups)
	if e != nil {
		return e
	}
	moved := []Doc{}
	supported := map[string]bool{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			rows.Close()
			return e
		}
		var raw Doc
		if e = json.Unmarshal(b, &raw); e != nil {
			rows.Close()
			return e
		}
		if str(raw["storyId"]) == str(left["id"]) && exclusive[str(raw["url"])] {
			moved = append(moved, raw)
		} else {
			for _, fp := range materialFingerprints(str(raw["sourceId"]), raw, str(raw["url"])) {
				supported[fp] = true
			}
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	fingerprints := map[string]bool{}
	for _, raw := range array(merge["fingerprints"]) {
		fingerprints[str(raw)] = true
	}
	for _, raw := range moved {
		for _, fp := range materialFingerprints(str(raw["sourceId"]), raw, str(raw["url"])) {
			fingerprints[fp] = true
		}
		raw["storyId"] = right["id"]
		if e = tx.Put(ctx, "raw_item", raw); e != nil {
			return e
		}
	}
	for fp := range fingerprints {
		if !supported[fp] {
			if _, e = tx.Exec(ctx, "UPDATE creata_fingerprints SET story_id=$1 WHERE fingerprint=$2 AND story_id=$3", right["id"], fp, left["id"]); e != nil {
				return e
			}
		}
	}
	return nil
}
