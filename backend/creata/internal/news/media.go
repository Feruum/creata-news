package news

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) mediaRoutes() {
	s.handle("GET /legacy-media/{path...}", func(w http.ResponseWriter, r *http.Request) error {
		legacy := "/" + r.PathValue("path")
		if !strings.HasPrefix(legacy, "/news/photos/") && !strings.HasPrefix(legacy, "/newspaper/issues/") {
			return errStatus(404, "Legacy media not found")
		}
		rows, e := s.Store.Pool.Query(r.Context(), "SELECT data->>'url' FROM creata_records WHERE kind='media' AND data->>'legacyPath'=$1 LIMIT 1", legacy)
		if e != nil {
			return e
		}
		defer rows.Close()
		if !rows.Next() {
			return errStatus(404, "Media not imported")
		}
		var target string
		if e = rows.Scan(&target); e != nil {
			return e
		}
		http.Redirect(w, r, target, http.StatusTemporaryRedirect)
		return nil
	})
	s.handle("POST "+base+"/admin/media", s.uploadMedia)
	s.handle("GET "+base+"/admin/media", s.listMedia)
	s.handle("PATCH "+base+"/admin/media/{id}", s.updateMedia)
	s.handle("DELETE "+base+"/admin/media/{id}", s.deleteMedia)
	s.handle("POST "+base+"/admin/media/cleanup", s.cleanupMedia)
	if s.Config.Environment != "production" && s.Config.S3Endpoint == "" {
		s.mux.Handle("GET /media/", http.StripPrefix("/media/", http.FileServer(http.Dir(s.Config.MediaDir))))
	}
}
func (s *Server) uploadMedia(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !has(u["roles"], "author") && !has(u["roles"], "editor") && !has(u["roles"], "chief_editor") {
		return errStatus(403, "Media upload denied")
	}
	if e = r.ParseMultipartForm(10 << 20); e != nil {
		return errStatus(400, "Media must be smaller than 10 MiB")
	}
	defer r.MultipartForm.RemoveAll()
	if r.FormValue("credit") == "" || r.FormValue("rights") == "" {
		return errStatus(400, "Media credit and rights are required")
	}
	f, _, e := r.FormFile("file")
	if e != nil {
		return errStatus(400, "Choose an image or MP4 video")
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, (10<<20)+1))
	if e != nil {
		return e
	}
	if len(data) > 10<<20 {
		return errStatus(413, "Media exceeds 10 MiB")
	}
	mime := http.DetectContentType(data)
	ext := imageExtension(mime)
	folder := "images/"
	if mime == "video/mp4" {
		ext = ".mp4"
		folder = "videos/"
	}
	if ext == "" {
		return errStatus(400, "Only JPEG, PNG, WebP, GIF and MP4 media are supported")
	}
	key := folder + id() + ext
	reservationID, e := s.reserveMediaObject(r.Context(), key)
	if e != nil {
		return e
	}
	location := ""
	if s.Config.S3Endpoint != "" {
		location, e = s.s3Put(r, key, mime, data)
	} else {
		if s.Config.Environment == "production" {
			return errStatus(503, "Media storage is not configured")
		}
		target := filepath.Join(s.Config.MediaDir, filepath.FromSlash(key))
		if e = os.MkdirAll(filepath.Dir(target), 0700); e == nil {
			e = os.WriteFile(target, data, 0600)
		}
		location = strings.TrimRight(s.Config.APIURL, "/") + "/media/" + key
	}
	if e != nil {
		return e
	}
	out := Doc{"id": id(), "url": location, "key": key, "mime": mime, "bytes": len(data), "revision": 1, "creatorId": u["id"], "credit": r.FormValue("credit"), "rights": r.FormValue("rights"), "caption": r.FormValue("caption"), "createdAt": now()}
	inspectSocialMedia(out, data)
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		if e := t.Put(r.Context(), "media", out); e != nil {
			return e
		}
		if e := t.Delete(r.Context(), "media_cleanup", reservationID); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "media.upload", str(out["id"]), Doc{"url": location})
	})
	if e != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if cleanupErr := s.removeMediaObject(cleanupCtx, key); cleanupErr != nil {
			slog.Error("media compensation failed", "key", key, "error", cleanupErr)
			_ = s.Store.Write(cleanupCtx, func(t *Tx) error {
				return t.Put(cleanupCtx, "media_cleanup", Doc{"id": id(), "key": key, "createdAt": now(), "error": cleanupErr.Error()})
			})
		} else {
			_ = s.Store.Write(cleanupCtx, func(t *Tx) error { return t.Delete(cleanupCtx, "media_cleanup", reservationID) })
		}
		return e
	}
	respond(w, 201, out)
	return nil
}
func hmacBytes(key []byte, value string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(value))
	return h.Sum(nil)
}
func (s *Server) s3Put(r *http.Request, key, mime string, data []byte) (string, error) {
	return s.s3Object(r.Context(), http.MethodPut, key, mime, data)
}
func (s *Server) s3Object(ctx context.Context, method, key, mime string, data []byte) (string, error) {
	c := s.Config
	if c.S3Bucket == "" || c.S3AccessKey == "" || c.S3SecretKey == "" {
		return "", errStatus(503, "S3 storage is not configured")
	}
	u, e := url.Parse(strings.TrimRight(c.S3Endpoint, "/") + "/" + c.S3Bucket + "/" + key)
	if e != nil {
		return "", e
	}
	t := time.Now().UTC()
	amz, date := t.Format("20060102T150405Z"), t.Format("20060102")
	bodyHash := sha256.Sum256(data)
	payload := hex.EncodeToString(bodyHash[:])
	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := "content-type:" + mime + "\nhost:" + u.Host + "\nx-amz-content-sha256:" + payload + "\nx-amz-date:" + amz + "\n"
	canonicalRequest := method + "\n" + u.EscapedPath() + "\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + payload
	requestHash := sha256.Sum256([]byte(canonicalRequest))
	scope := date + "/" + c.S3Region + "/s3/aws4_request"
	toSign := "AWS4-HMAC-SHA256\n" + amz + "\n" + scope + "\n" + hex.EncodeToString(requestHash[:])
	dateKey := hmacBytes([]byte("AWS4"+c.S3SecretKey), date)
	regionKey := hmacBytes(dateKey, c.S3Region)
	serviceKey := hmacBytes(regionKey, "s3")
	signingKey := hmacBytes(serviceKey, "aws4_request")
	signature := hex.EncodeToString(hmacBytes(signingKey, toSign))
	req, e := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(data))
	if e != nil {
		return "", e
	}
	req.Header.Set("Content-Type", mime)
	req.Header.Set("X-Amz-Date", amz)
	req.Header.Set("X-Amz-Content-Sha256", payload)
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", c.S3AccessKey, scope, signedHeaders, signature))
	res, e := s.client.Do(req)
	if e != nil {
		return "", e
	}
	defer res.Body.Close()
	if (res.StatusCode < 200 || res.StatusCode >= 300) && !(method == http.MethodDelete && res.StatusCode == 404) {
		return "", fmt.Errorf("S3 upload returned %d", res.StatusCode)
	}
	public := os.Getenv("CREATA_MEDIA_PUBLIC_URL")
	if public != "" {
		return strings.TrimRight(public, "/") + "/" + key, nil
	}
	return u.String(), nil
}
