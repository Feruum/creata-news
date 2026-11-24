package news

import (
	_ "embed"
	"net/http"
	"strconv"
	"strings"
)

//go:embed public_catalog_cache.sql
var publicCatalogSchema string

// The catalog changes in the same database transaction as each published
// snapshot, so all API replicas can validate a response after a publish or withdrawal.
func (s *Server) catalogResponse(fn endpoint) endpoint {
	return func(w http.ResponseWriter, r *http.Request) error {
		var epoch string
		var version int64
		if err := s.Store.Pool.QueryRow(r.Context(),
			"SELECT epoch, version FROM creata_article_catalog_generation WHERE singleton").Scan(&epoch, &version); err != nil {
			return err
		}
		tag := `W/"` + hash("article-catalog-v1:"+epoch+":"+strconv.FormatInt(version, 10)+":"+r.URL.RequestURI()) + `"`
		w.Header().Set("ETag", tag)
		for _, value := range strings.Split(r.Header.Get("If-None-Match"), ",") {
			if strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "W/")) == strings.TrimPrefix(tag, "W/") {
				w.WriteHeader(http.StatusNotModified)
				return nil
			}
		}
		if err := fn(w, r); err != nil {
			w.Header().Del("ETag")
			return err
		}
		return nil
	}
}
