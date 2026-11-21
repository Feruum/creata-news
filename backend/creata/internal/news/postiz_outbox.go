package news

import (
	"bytes"
	"context"
	"errors"
	"image/jpeg"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Record byte-derived metadata when an asset enters the library. The social
// worker accepts only a library record matching the approved article's asset.
func inspectSocialMedia(d Doc, data []byte) {
	switch http.DetectContentType(data) {
	case "image/jpeg":
		if dimensions, e := jpeg.DecodeConfig(bytes.NewReader(data)); e == nil {
			d["width"], d["height"] = dimensions.Width, dimensions.Height
		}
	case "video/mp4":
		d["verifiedMp4"] = true
	}
}

func publicMediaURL(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Hostname() == "" || isLoopback(u.Hostname()) {
		return false
	}
	ip := net.ParseIP(u.Hostname())
	return ip == nil || (ip.IsGlobalUnicast() && !ip.IsPrivate())
}

func socialMedia(ctx context.Context, t *Tx, a Doc, target postizTarget) (string, string, error) {
	provider := target.channel.Provider
	want := ""
	switch provider {
	case "instagram", "instagram-standalone":
		want = "Instagram needs a rights-reviewed library JPEG up to 8 MiB, with a public HTTPS URL and a 4:5 to 1.91:1 aspect ratio"
	case "youtube":
		want = "YouTube needs a rights-reviewed library MP4 with a public HTTPS URL"
	default:
		return "", "", nil
	}
	raw := str(object(a["media"])["url"])
	if target.mediaID == "" && !publicMediaURL(raw) {
		return "", want, nil
	}
	var m Doc
	var e error
	if target.mediaID != "" {
		e = t.QueryRow(ctx, "SELECT data FROM creata_records WHERE kind='media' AND id=$1", target.mediaID).Scan(&m)
	} else {
		e = t.QueryRow(ctx, "SELECT data FROM creata_records WHERE kind='media' AND data->>'url'=$1 LIMIT 1", raw).Scan(&m)
	}
	if errors.Is(e, pgx.ErrNoRows) {
		return "", want, nil
	}
	if e != nil {
		return "", "", e
	}
	raw = str(m["url"])
	if !publicMediaURL(raw) || boolean(m["needsRightsReview"]) || strings.TrimSpace(str(m["credit"])) == "" || strings.TrimSpace(str(m["rights"])) == "" || integer(m["bytes"]) <= 0 {
		return "", want, nil
	}
	if provider == "youtube" {
		if m["mime"] != "video/mp4" || !boolean(m["verifiedMp4"]) {
			return "", want, nil
		}
	} else {
		w, h := integer(m["width"]), integer(m["height"])
		if m["mime"] != "image/jpeg" || integer(m["bytes"]) > 8<<20 || w <= 0 || h <= 0 || w*5 < h*4 || w*100 > h*191 {
			return "", want, nil
		}
	}
	return raw, "", nil
}

func cancelScheduledPostiz(t *Tx, ctx context.Context, aid string) error {
	_, e := t.Exec(ctx, "UPDATE creata_postiz_delivery SET status='cancelled',updated_at=now() WHERE article_id=$1 AND status='scheduled'", aid)
	return e
}

func (s *Server) stagePostiz(t *Tx, ctx context.Context, a Doc, targets []postizTarget) error {
	if e := cancelScheduledPostiz(t, ctx, str(a["id"])); e != nil {
		return e
	}
	for _, target := range targets {
		c, lang := target.channel, target.lang
		content, link, e := s.postizText(a, c.Provider, lang)
		if e != nil {
			return e
		}
		mediaURL, mediaError, e := socialMedia(ctx, t, a, target)
		if e != nil {
			return e
		}
		settings := Doc{"__type": c.Provider}
		switch c.Provider {
		case "x":
			settings["who_can_reply_post"] = "everyone"
		case "facebook":
			settings["url"] = link
		case "instagram", "instagram-standalone":
			settings["post_type"] = "post"
		case "youtube":
			title := []rune(strings.Join(strings.Fields(str(object(a["title"])[lang])), " "))
			if len(title) < 2 {
				title = append(title, []rune(" | Creata News")...)
			}
			settings["title"] = string(title[:min(100, len(title))])
			settings["type"] = target.visibility
		}
		snap := Doc{"channelName": c.Name, "provider": c.Provider, "content": content, "articleUrl": link, "mode": s.postiz.mode, "settings": settings, "mediaUrl": mediaURL, "mediaError": mediaError}
		if target.mediaID != "" {
			snap["mediaId"] = target.mediaID
		}
		if target.visibility != "" {
			snap["visibility"] = target.visibility
		}
		_, e = t.Exec(ctx, `INSERT INTO creata_postiz_delivery(id,article_id,revision,channel_id,locale,status,snapshot)
			VALUES ($1,$2,$3,$4,$5,'scheduled',$6) ON CONFLICT(article_id,revision,channel_id,locale)
			DO UPDATE SET status='scheduled',snapshot=excluded.snapshot,error='',next_at=now(),updated_at=now()
			WHERE creata_postiz_delivery.status='cancelled' AND creata_postiz_delivery.attempts=0`,
			id(), a["id"], integer(a["revision"]), c.ID, lang, snap)
		if e != nil {
			return e
		}
	}
	return nil
}

// Seed and content import intentionally call only publish(). Only the editorial
// action and due scheduler may activate the selected revision's deliveries.
func publishEditorial(t *Tx, ctx context.Context, a Doc, note string) error {
	if e := publish(t, ctx, a, note); e != nil {
		return e
	}
	if _, e := t.Exec(ctx, `UPDATE creata_postiz_delivery SET status='cancelled',updated_at=now()
		WHERE article_id=$1 AND revision<>$2 AND status IN ('scheduled','queued')`, a["id"], integer(a["revision"])); e != nil {
		return e
	}
	_, e := t.Exec(ctx, `UPDATE creata_postiz_delivery SET status=CASE WHEN snapshot->>'mediaError'='' THEN 'queued' ELSE 'missing_media' END,
		error=snapshot->>'mediaError',next_at=now(),updated_at=now() WHERE article_id=$1 AND revision=$2 AND status='scheduled'`, a["id"], integer(a["revision"]))
	return e
}

func (s *Server) postizHistory(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	aid := r.PathValue("id")
	a, e := s.Store.Get(r.Context(), "article", aid)
	if e != nil {
		return e
	}
	if !canEdit(u, a) && !has(u["roles"], "admin") {
		return errStatus(403, "Article access denied")
	}
	rows, e := s.Store.Pool.Query(r.Context(), `SELECT (snapshot - 'settings' - 'mediaUrl' - 'mediaError') ||
		jsonb_strip_nulls(jsonb_build_object('id',id,'channelId',channel_id,'revision',revision,'locale',locale,'status',status,
		'attempts',attempts,'postId',nullif(post_id,''),'releaseUrl',nullif(release_url,''),'error',nullif(error,''),'updatedAt',updated_at))
		FROM creata_postiz_delivery WHERE article_id=$1 ORDER BY revision DESC,channel_id,locale`, aid)
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []Doc{}
	for rows.Next() {
		var item Doc
		if e = rows.Scan(&item); e != nil {
			return e
		}
		items = append(items, item)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	respond(w, 200, Doc{"items": items})
	return nil
}
