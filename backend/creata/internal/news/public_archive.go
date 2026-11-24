package news

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const (
	maxArticlePageSize = 100
	// These Unicode characters match JavaScript's trim() and \s in the reader.
	archiveWhitespace = `\0009\000A\000B\000C\000D\0020\00A0\1680\2000\2001\2002\2003\2004\2005\2006\2007\2008\2009\200A\2028\2029\202F\205F\3000\FEFF`
	publicAuthorJoin  = `LEFT JOIN creata_records author ON author.kind = 'author' AND author.id = p.data ->> 'authorId'`
)

type articleArchivePage struct {
	Items      []Doc  `json:"items"`
	Total      *int   `json:"total,omitempty"`
	NextCursor string `json:"nextCursor,omitempty"`
}

type articleArchiveCursor struct {
	createdAt time.Time
	id        string
}

func (c articleArchiveCursor) encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.createdAt.Format(time.RFC3339Nano) + "\n" + c.id))
}

func (s *Server) articlesPublic(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	var limit int
	if raw := q.Get("limit"); raw != "" {
		var err error
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxArticlePageSize {
			return errStatus(400, "Choose an article limit between 1 and 100")
		}
	}
	var cursor articleArchiveCursor
	if raw := q.Get("cursor"); raw != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		stamp, recordID, ok := strings.Cut(string(decoded), "\n")
		cursor = articleArchiveCursor{createdAt: parseTime(stamp), id: recordID}
		if limit == 0 || err != nil || !ok || cursor.createdAt.IsZero() || cursor.id == "" || !utf8.ValidString(cursor.id) || strings.ContainsRune(cursor.id, 0) {
			return errStatus(400, "Invalid article cursor")
		}
	}
	page, err := s.Store.articleArchive(r.Context(), q, limit, cursor)
	if err != nil {
		return err
	}
	respond(w, 200, page)
	return nil
}

// articleArchiveText returns the locale's nonempty string with the same Russian fallback as localize.
func articleArchiveText(record, field, locale string) string {
	text := func(lang string) string {
		value := record + " -> '" + field + "' -> '" + lang + "'"
		return "(CASE WHEN jsonb_typeof(" + value + ") = 'string' THEN " + value + " #>> '{}' END)"
	}
	if locale == "ru" {
		return "COALESCE(" + text("ru") + ", '')"
	}
	return "COALESCE(NULLIF(" + text(locale) + ", ''), " + text("ru") + ", '')"
}

func (s *Store) articleArchive(ctx context.Context, q url.Values, limit int, cursor articleArchiveCursor) (articleArchivePage, error) {
	result := articleArchivePage{Items: []Doc{}}
	var args []any
	param := func(value any) string {
		args = append(args, value)
		return "$" + strconv.Itoa(len(args))
	}
	where := "kind = 'publication'"
	for _, filter := range []struct{ query, field string }{
		{"layer", "layer"}, {"category", "categoryId"}, {"authorId", "authorId"}, {"region", "region"},
	} {
		if value := q.Get(filter.query); value != "" {
			where += " AND data ->> '" + filter.field + "' = " + param(value) +
				" AND jsonb_typeof(data -> '" + filter.field + "') = 'string'"
		}
	}
	if format := q.Get("format"); format != "" {
		where += ` AND COALESCE(NULLIF(CASE WHEN jsonb_typeof(data -> 'format') = 'string' THEN data ->> 'format' END, ''), 'article') = ` + param(format)
	}
	lang := locale(q.Get("locale"))
	if q.Get("translated") == "1" {
		// A translation must exist in this locale, regardless of the search fallback.
		title := "data -> 'title' -> '" + lang + "'"
		where += " AND jsonb_typeof(" + title + ") = 'string' AND btrim(" + title + " #>> '{}', U&'" + archiveWhitespace + "') <> ''"
	}
	if search := q.Get("q"); search != "" {
		title := articleArchiveText("data", "title", lang)
		body := articleArchiveText("data", "body", lang)
		where += ` AND strpos(lower((` + title + ` || ' ' || ` + body + `) COLLATE pg_catalog."und-x-icu"), lower(` + param(search) +
			`::text COLLATE pg_catalog."und-x-icu")) > 0`
	}

	projection := "p.data"
	if q.Get("view") == "summary" {
		minutes := func(lang string) string {
			body := articleArchiveText("p.data", "body", lang)
			return "GREATEST(1, (regexp_count(" + body + ", U&'[^" + archiveWhitespace + "]+') + 219) / 220)"
		}
		projection = "(p.data - 'body') || jsonb_build_object('readingMinutes', jsonb_build_object('ru', " +
			minutes("ru") + ", 'kk', " + minutes("kk") + ", 'en', " + minutes("en") + "))"
	}

	page := "SELECT data, created_at, id FROM creata_records WHERE " + where
	if limit > 0 {
		if !cursor.createdAt.IsZero() {
			stamp, recordID := param(cursor.createdAt), param(cursor.id)
			page += " AND created_at <= " + stamp + " AND (created_at < " + stamp + " OR id > " + recordID + ")"
		}
		page += " ORDER BY created_at DESC, id LIMIT " + param(limit+1)
	}
	from := " FROM (" + page + ") p "
	totalProjection := "0::bigint"
	includeTotal := limit == 0 || q.Get("includeTotal") != "0"
	if limit > 0 && includeTotal {
		from = " FROM (SELECT count(*) AS total FROM creata_records WHERE " + where + ") matching LEFT JOIN (" + page + ") p ON true "
		totalProjection = "matching.total"
	}
	rows, err := s.Pool.Query(ctx, "SELECT "+projection+", author.data, p.created_at, p.id, "+totalProjection+from+publicAuthorJoin+
		" ORDER BY p.created_at DESC, p.id", args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	var last articleArchiveCursor
	var total int
	for rows.Next() {
		var article, author Doc
		var createdAt *time.Time
		var recordID *string
		if err := rows.Scan(&article, &author, &createdAt, &recordID, &total); err != nil {
			return result, err
		}
		if recordID == nil {
			// The matching count still returns one row when the page is empty.
			break
		}
		if limit > 0 && len(result.Items) == limit {
			result.NextCursor = last.encode()
			break
		}
		result.Items = append(result.Items, publicArticleWithAuthor(article, author))
		last = articleArchiveCursor{createdAt: *createdAt, id: *recordID}
	}
	if limit == 0 {
		total = len(result.Items)
	}
	if includeTotal {
		result.Total = &total
	}
	return result, rows.Err()
}

func publicArticleWithAuthor(article, author Doc) Doc {
	public := publicArticle(article)
	if author != nil {
		public["author"] = author
	}
	return public
}

func (s *Server) articlePublic(w http.ResponseWriter, r *http.Request) error {
	var article, author Doc
	var corrections []Doc
	err := s.Store.Pool.QueryRow(r.Context(), `
		SELECT p.data, author.data, COALESCE((
			SELECT jsonb_agg(jsonb_build_object(
				'revision', v.data -> 'revision', 'note', v.data -> 'note', 'publishedAt', v.data -> 'publishedAt'
			) ORDER BY v.created_at DESC, v.id)
			FROM creata_records v
			WHERE v.kind = 'publication_version' AND v.data ->> 'articleId' = p.id
				AND jsonb_typeof(v.data -> 'note') = 'string' AND v.data ->> 'note' <> ''
		), '[]'::jsonb)
		FROM creata_records p `+publicAuthorJoin+`
		WHERE p.kind = 'publication' AND p.data ->> 'slug' = $1 AND jsonb_typeof(p.data -> 'slug') = 'string'
		ORDER BY p.created_at DESC, p.id
		LIMIT 1`, r.PathValue("slug")).Scan(&article, &author, &corrections)
	if errors.Is(err, pgx.ErrNoRows) {
		return errStatus(404, "Article not found")
	}
	if err != nil {
		return err
	}
	public := publicArticleWithAuthor(article, author)
	public["corrections"] = corrections
	respond(w, 200, public)
	return nil
}

func (s *Server) regionsPublic(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.Store.Pool.Query(r.Context(), `
		SELECT region FROM (
			SELECT DISTINCT ON (data ->> 'region') data ->> 'region' AS region, created_at, id
			FROM creata_records
			WHERE kind = 'publication' AND jsonb_typeof(data -> 'region') = 'string' AND data ->> 'region' <> ''
			ORDER BY data ->> 'region', created_at DESC, id
		) recent_regions ORDER BY created_at DESC, id LIMIT 12`)
	if err != nil {
		return err
	}
	regions, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	respond(w, 200, Doc{"items": regions})
	return nil
}

func (s *Store) publicOpportunityCandidates(ctx context.Context, ids []string) ([]Doc, error) {
	if len(ids) == 0 {
		return s.List(ctx, "opportunity")
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT data FROM creata_records
		WHERE kind = 'opportunity' AND id = ANY($1)
		ORDER BY created_at DESC, id`, ids)
	if err != nil {
		return nil, fmt.Errorf("load opportunities by ID: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowTo[Doc])
}
