package news

import (
	"encoding/json"
	"net/http"
)

// Rank only the public snapshots by accepted view events in the last seven days.
// Drafts, withdrawn publications and untranslated stories cannot enter this feed.
func (s *Server) popularPublic(w http.ResponseWriter, r *http.Request) error {
	lang := locale(r.URL.Query().Get("locale"))
	rows, err := s.Store.Pool.Query(r.Context(), `
		SELECT publication.data FROM creata_records publication
		JOIN (
			SELECT target_id, count(*) AS views FROM creata_events
			WHERE kind='view' AND created_at >= now() - interval '7 days'
			GROUP BY target_id
		) popularity ON popularity.target_id=publication.id
		WHERE publication.kind='publication'
		  AND publication.data->>'status'='published'
		  AND trim(coalesce(publication.data->'title'->>$1,''))<>''
		ORDER BY popularity.views DESC, publication.data->>'publishedAt' DESC, publication.id
		LIMIT 6`, lang)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []Doc{}
	for rows.Next() {
		var raw []byte
		var article Doc
		if err = rows.Scan(&raw); err != nil {
			return err
		}
		if err = json.Unmarshal(raw, &article); err != nil {
			return err
		}
		public := publicArticle(article)
		delete(public, "reviewNotes")
		items = append(items, s.decorate(r.Context(), public))
	}
	if err = rows.Err(); err != nil {
		return err
	}
	respond(w, http.StatusOK, Doc{"items": items, "total": len(items)})
	return nil
}
