package news

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func imageExtension(mime string) string {
	return map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif"}[mime]
}
func (s *Server) reserveMediaObject(ctx context.Context, key string) (string, error) {
	reservationID := "upload-" + hash(key)
	e := s.Store.Write(ctx, func(t *Tx) error {
		return t.Put(ctx, "media_cleanup", Doc{"id": reservationID, "key": key, "createdAt": now(), "eligibleAfter": time.Now().Add(time.Hour).UTC().Format(time.RFC3339), "reason": "Incomplete media upload"})
	})
	return reservationID, e
}
func (s *Server) mediaReferences(ctx context.Context, url, id string) ([]Doc, error) {
	media, e := s.Store.Get(ctx, "media", id)
	if e != nil {
		return nil, e
	}
	legacy := str(media["legacyPath"])
	rows, e := s.Store.Pool.Query(ctx, `SELECT kind,id FROM creata_records WHERE kind NOT IN ('media','media_cleanup','audit')
		AND (strpos(data::text,$1)>0 OR strpos(data::text,$2)>0 OR ($3<>'' AND strpos(data::text,$3)>0))
		UNION ALL SELECT 'postiz_delivery',id FROM creata_postiz_delivery WHERE status IN ('scheduled','queued','sending') AND snapshot->>'mediaUrl'=$1
		ORDER BY 1,2`, url, "\""+id+"\"", legacy)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Doc{}
	for rows.Next() {
		var k, i string
		if e = rows.Scan(&k, &i); e != nil {
			return nil, e
		}
		out = append(out, Doc{"kind": k, "id": i})
	}
	return out, rows.Err()
}
func (s *Server) listMedia(w http.ResponseWriter, r *http.Request) error {
	if _, e := s.require(r, true); e != nil {
		return e
	}
	all, e := s.Store.List(r.Context(), "media")
	if e != nil {
		return e
	}
	q := strings.ToLower(r.URL.Query().Get("q"))
	// Read reference-bearing documents once; large imported issues must not be
	// decompressed by PostgreSQL once for every image in the library.
	rows, e := s.Store.Pool.Query(r.Context(), "SELECT kind,id,data FROM creata_records WHERE kind NOT IN ('media','media_cleanup','audit')")
	if e != nil {
		return e
	}
	type referenceDoc struct {
		kind, id string
		raw      []byte
	}
	documents := []referenceDoc{}
	for rows.Next() {
		var doc referenceDoc
		if e = rows.Scan(&doc.kind, &doc.id, &doc.raw); e != nil {
			rows.Close()
			return e
		}
		documents = append(documents, doc)
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return e
	}
	items := []Doc{}
	for _, d := range all {
		if q != "" && !strings.Contains(strings.ToLower(fmt.Sprint(d["caption"], d["credit"], d["legacyPath"], d["url"])), q) {
			continue
		}
		refs := []Doc{}
		urlBytes, idBytes, legacyBytes := []byte(str(d["url"])), []byte("\""+str(d["id"])+"\""), []byte(str(d["legacyPath"]))
		for _, doc := range documents {
			if bytes.Contains(doc.raw, urlBytes) || bytes.Contains(doc.raw, idBytes) || len(legacyBytes) > 0 && bytes.Contains(doc.raw, legacyBytes) {
				refs = append(refs, Doc{"kind": doc.kind, "id": doc.id})
			}
		}
		d["references"] = refs
		d["referenced"] = len(refs) > 0
		items = append(items, d)
	}
	respond(w, 200, Doc{"items": items, "total": len(items), "acceptedTypes": []string{"image/jpeg", "image/png", "image/webp", "image/gif", "video/mp4"}})
	return nil
}
func (s *Server) updateMedia(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !has(u["roles"], "editor") && !newspaperPublisher(u) {
		return errStatus(403, "Only an editor may change media rights")
	}
	in, e := decode(r)
	if e != nil {
		return e
	}
	var out Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		d, e := t.Get(r.Context(), "media", r.PathValue("id"))
		if e != nil {
			return e
		}
		if integer(in["expectedRevision"]) != integer(d["revision"]) {
			return errStatus(409, "Media changed; reload before saving")
		}
		for _, key := range []string{"caption", "credit", "rights"} {
			if v, ok := in[key]; ok {
				if len(str(v)) > 8000 {
					return errStatus(400, "Media metadata is too long")
				}
				d[key] = v
			}
		}
		if !nonempty(d["credit"]) || !nonempty(d["rights"]) {
			return errStatus(400, "Credit and rights are required")
		}
		if boolean(in["confirmRights"]) {
			d["needsRightsReview"] = false
			d["rightsReviewedBy"] = u["id"]
			d["rightsReviewedAt"] = now()
		}
		d["revision"] = integer(d["revision"]) + 1
		out = d
		if e = t.Put(r.Context(), "media", d); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "media.update", str(d["id"]), Doc{"revision": d["revision"]})
	})
	if e != nil {
		return e
	}
	respond(w, 200, out)
	return nil
}
func (s *Server) removeMediaObject(ctx context.Context, key string) error {
	if key == "" || strings.Contains(key, "..") || filepath.IsAbs(key) || strings.Contains(key, "\\") {
		return errStatus(400, "Unsafe media object key")
	}
	if s.Config.S3Endpoint != "" {
		_, e := s.s3Object(ctx, http.MethodDelete, key, "application/octet-stream", nil)
		return e
	}
	root, e := filepath.Abs(s.Config.MediaDir)
	if e != nil {
		return e
	}
	target, e := filepath.Abs(filepath.Join(root, filepath.FromSlash(key)))
	if e != nil {
		return e
	}
	if !strings.HasPrefix(target, root+string(filepath.Separator)) {
		return errStatus(400, "Unsafe media target")
	}
	e = os.Remove(target)
	if os.IsNotExist(e) {
		return nil
	}
	return e
}
func (s *Server) deleteMedia(w http.ResponseWriter, r *http.Request) error {
	var objectKey string
	cleanupID := "delete-" + id()
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !newspaperPublisher(u) {
		return errStatus(403, "Only a chief editor or administrator may delete media")
	}
	in, e := decode(r)
	if e != nil {
		return e
	}
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		d, e := t.Get(r.Context(), "media", r.PathValue("id"))
		if e != nil {
			return e
		}
		if integer(in["expectedRevision"]) != integer(d["revision"]) {
			return errStatus(409, "Media changed; reload")
		}
		refs, e := s.mediaReferences(r.Context(), str(d["url"]), str(d["id"]))
		if e != nil {
			return e
		}
		if len(refs) > 0 {
			return errStatus(409, "Media is referenced by content or its revision history")
		}
		if str(d["key"]) == "" {
			return errStatus(409, "Legacy object key is unknown; storage deletion is unavailable")
		}
		objectKey = str(d["key"])
		if e = t.Put(r.Context(), "media_cleanup", Doc{"id": cleanupID, "key": objectKey, "createdAt": now(), "reason": "Deleted unreferenced media"}); e != nil {
			return e
		}
		if e = t.Delete(r.Context(), "media", str(d["id"])); e != nil {
			return e
		}
		if str(d["legacyPath"]) != "" {
			if e = t.Put(r.Context(), "media_tombstone", Doc{"id": d["id"], "legacyPath": d["legacyPath"], "deletedAt": now(), "actorId": u["id"]}); e != nil {
				return e
			}
		}
		return t.Audit(r.Context(), u, "media.delete", str(d["id"]), nil)
	})
	if e != nil {
		return e
	}
	pending := false
	if e = s.removeMediaObject(r.Context(), objectKey); e != nil {
		pending = true
	} else {
		e = s.Store.Write(r.Context(), func(t *Tx) error { return t.Delete(r.Context(), "media_cleanup", cleanupID) })
		pending = e != nil
	}
	respond(w, 200, Doc{"deleted": true, "pendingCleanup": pending})
	return nil
}
func (s *Server) cleanupMedia(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !newspaperPublisher(u) {
		return errStatus(403, "Only a chief editor or administrator may clean media")
	}
	all, e := s.Store.List(r.Context(), "media_cleanup")
	if e != nil {
		return e
	}
	done := 0
	for _, d := range all {
		if parseTime(d["eligibleAfter"]).After(time.Now()) {
			continue
		}
		if e = s.removeMediaObject(r.Context(), str(d["key"])); e != nil {
			continue
		}
		if e = s.Store.Write(r.Context(), func(t *Tx) error { return t.Delete(r.Context(), "media_cleanup", str(d["id"])) }); e != nil {
			return e
		}
		done++
	}
	respond(w, 200, Doc{"removed": done, "remaining": len(all) - done})
	return nil
}

// Import objects by deterministic key, then rewrite persisted documents. A retry
// reuses records and object keys; media is never fetched from arbitrary URLs.
func (s *Server) importContentAssets(ctx context.Context, b []byte, assetDir string) ([]byte, error) {
	root, e := filepath.Abs(assetDir)
	if e != nil {
		return nil, e
	}
	replacements := map[string]string{}
	existing, e := s.Store.List(ctx, "media")
	if e != nil {
		return nil, e
	}
	byPath := map[string]Doc{}
	deleted, e := s.Store.List(ctx, "media_tombstone")
	if e != nil {
		return nil, e
	}
	deletedPaths := map[string]bool{}
	for _, d := range deleted {
		deletedPaths[str(d["legacyPath"])] = true
	}
	urlUpdates := []Doc{}
	metadataUpdates := []Doc{}
	for _, d := range existing {
		// Local development ports may change. Keep persisted references pointing
		// to the current API without changing CDN URLs or external source links.
		if s.Config.S3Endpoint == "" {
			if old, e := url.Parse(str(d["url"])); e == nil && isLoopback(old.Hostname()) && strings.HasPrefix(old.Path, "/media/") {
				key := strings.TrimPrefix(old.Path, "/media/")
				if !strings.Contains(key, "..") && !strings.Contains(key, "\\") {
					next := strings.TrimRight(s.Config.APIURL, "/") + "/media/" + key
					if next != str(d["url"]) {
						replacements[str(d["url"])] = next
						d["url"] = next
						d["key"] = key
						urlUpdates = append(urlUpdates, d)
					}
				}
			}
		}
		if p := str(d["legacyPath"]); p != "" {
			byPath[p] = d
		}
	}
	e = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in media import: %s", path)
		}
		relative, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		legacy := "/" + filepath.ToSlash(relative)
		if deletedPaths[legacy] {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !has([]string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp4", ".webm", ".pdf"}, ext) {
			return nil
		}
		if old := byPath[legacy]; old != nil {
			// Older imports predate dimensions and MP4 verification. Inspect the
			// bundled bytes only when they still match the stored content key.
			if (old["mime"] == "image/jpeg" && integer(old["width"]) == 0) || (old["mime"] == "video/mp4" && !boolean(old["verifiedMp4"])) {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if old["key"] == "imports/"+hash(legacy+":"+hash(string(data)))+ext {
					metadata := Doc{"id": old["id"]}
					inspectSocialMedia(metadata, data)
					if len(metadata) > 1 {
						metadataUpdates = append(metadataUpdates, metadata)
					}
				}
			}
			replacements[legacy] = str(old["url"])
			if legacy == "/newspaper/issues/01/creata-news-01.pdf" {
				replacements["https://rjklqsevcrikzer4.public.blob.vercel-storage.com/issues/01/creata-news-01.pdf"] = str(old["url"])
			}
			return nil
		}
		data, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		key := "imports/" + hash(legacy+":"+hash(string(data))) + ext
		reservationID, e := s.reserveMediaObject(ctx, key)
		if e != nil {
			return e
		}
		location := ""
		mime := http.DetectContentType(data)
		if ext == ".mp4" {
			mime = "video/mp4"
		}
		if ext == ".webm" {
			mime = "video/webm"
		}
		if s.Config.S3Endpoint != "" {
			location, e = s.s3Object(ctx, http.MethodPut, key, mime, data)
		} else {
			target := filepath.Join(s.Config.MediaDir, filepath.FromSlash(key))
			if e = os.MkdirAll(filepath.Dir(target), 0700); e == nil {
				e = os.WriteFile(target, data, 0600)
			}
			location = strings.TrimRight(s.Config.APIURL, "/") + "/media/" + key
		}
		if e != nil {
			return e
		}
		d := Doc{"id": "import-" + hash(legacy), "url": location, "key": key, "mime": mime, "bytes": len(data), "legacyPath": legacy, "caption": legacy, "credit": "Imported Creata publication", "rights": "Legacy import: rights and photographer require editorial review", "needsRightsReview": true, "revision": 1, "createdAt": now()}
		inspectSocialMedia(d, data)
		if e = s.Store.Write(ctx, func(t *Tx) error {
			if e := t.Put(ctx, "media", d); e != nil {
				return e
			}
			return t.Delete(ctx, "media_cleanup", reservationID)
		}); e != nil {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			if cleanupErr := s.removeMediaObject(cleanupCtx, key); cleanupErr == nil {
				_ = s.Store.Write(cleanupCtx, func(t *Tx) error { return t.Delete(cleanupCtx, "media_cleanup", reservationID) })
			}
			return e
		}
		replacements[legacy] = location
		if legacy == "/newspaper/issues/01/creata-news-01.pdf" {
			replacements["https://rjklqsevcrikzer4.public.blob.vercel-storage.com/issues/01/creata-news-01.pdf"] = location
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	rewrite := func(raw []byte) []byte { return rewriteImportedJSON(raw, replacements) }
	e = s.Store.Write(ctx, func(t *Tx) error {
		rows, e := t.Query(ctx, "SELECT kind,id,data FROM creata_records WHERE kind NOT IN ('media','audit','media_cleanup')")
		if e != nil {
			return e
		}
		type record struct {
			k, id string
			b     []byte
		}
		list := []record{}
		for rows.Next() {
			var d record
			if e = rows.Scan(&d.k, &d.id, &d.b); e != nil {
				rows.Close()
				return e
			}
			list = append(list, d)
		}
		rows.Close()
		if e = rows.Err(); e != nil {
			return e
		}
		for _, d := range list {
			next := rewrite(d.b)
			if string(next) == string(d.b) {
				continue
			}
			var data Doc
			if e = json.Unmarshal(next, &data); e != nil {
				return e
			}
			if e = t.Put(ctx, d.k, data); e != nil {
				return e
			}
		}
		for _, metadata := range metadataUpdates {
			if _, e := t.Exec(ctx, "UPDATE creata_records SET data=data || $2::jsonb,updated_at=now() WHERE kind='media' AND id=$1", metadata["id"], metadata); e != nil {
				return e
			}
		}
		for _, d := range urlUpdates {
			current, getErr := t.Get(ctx, "media", str(d["id"]))
			if getErr != nil {
				var apiErr *apiError
				if errors.As(getErr, &apiErr) && apiErr.status == 404 {
					continue
				}
				return getErr
			}
			current["url"] = d["url"]
			current["key"] = d["key"]
			if e = t.Put(ctx, "media", current); e != nil {
				return e
			}
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	return rewrite(b), nil
}

func rewriteImportedJSON(raw []byte, replacements map[string]string) []byte {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return raw
	}
	var walk func(any) any
	changed := false
	walk = func(value any) any {
		switch v := value.(type) {
		case string:
			if next, ok := replacements[v]; ok {
				if next != v {
					changed = true
				}
				return next
			}
			if strings.Contains(v, "<") {
				for before, after := range replacements {
					for _, quote := range []string{"\"", "'"} {
						next := strings.ReplaceAll(v, quote+before+quote, quote+after+quote)
						if next != v {
							changed = true
						}
						v = next
					}
				}
			}
			return v
		case []any:
			for i, x := range v {
				v[i] = walk(x)
			}
			return v
		case map[string]any:
			for k, x := range v {
				v[k] = walk(x)
			}
			return v
		default:
			return v
		}
	}
	value = walk(value)
	if !changed {
		return raw
	}
	next, e := json.Marshal(value)
	if e != nil {
		return raw
	}
	return next
}
