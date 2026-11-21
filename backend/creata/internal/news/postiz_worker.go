package news

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type postizJob struct {
	id, channel, postID, status string
	snapshot                    Doc
	started                     time.Time
	attempts                    int
}

func (s *Server) RunPostiz(ctx context.Context) {
	if s.postiz.key == "" {
		return
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		if e := s.postizTick(ctx); e != nil && ctx.Err() == nil {
			slog.Error("Postiz worker tick", "error", e)
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) postizTick(ctx context.Context) error {
	if s.postiz.key == "" {
		return nil
	}
	// A withdrawal or correction may commit while a rate-limited create is
	// returning to the queue. Cancel that stale retry even after a restart.
	if _, e := s.Store.Pool.Exec(ctx, `UPDATE creata_postiz_delivery d SET status='cancelled',updated_at=now()
		WHERE status='queued' AND NOT EXISTS (SELECT 1 FROM creata_records p WHERE p.kind='publication' AND p.id=d.article_id AND (p.data->>'revision')::int=d.revision)`); e != nil {
		return e
	}
	// Commit the in-flight marker before the HTTP request. A worker that dies
	// after POST may already have created a post, so its replacement only reads.
	if _, e := s.Store.Pool.Exec(ctx, `UPDATE creata_postiz_delivery SET status='uncertain',error='Delivery was interrupted; checking Postiz before any resend',updated_at=now()
		WHERE status='sending' AND next_at<=now()`); e != nil {
		return e
	}
	var j postizJob
	e := s.Store.Pool.QueryRow(ctx, `WITH candidate AS (
		SELECT d.id FROM creata_postiz_delivery d JOIN creata_records p ON p.kind='publication' AND p.id=d.article_id AND (p.data->>'revision')::int=d.revision
		WHERE d.status='queued' AND d.next_at<=now() AND NOT EXISTS(SELECT 1 FROM creata_postiz_throttle WHERE scope='worker' AND until_at>now())
		ORDER BY d.next_at,d.id LIMIT 1 FOR UPDATE OF d SKIP LOCKED)
		UPDATE creata_postiz_delivery d SET status='sending',attempts=attempts+1,dispatched_at=now(),updated_at=now(),next_at=now()+interval '2 minutes'
		FROM candidate c WHERE d.id=c.id RETURNING d.id,d.channel_id,d.snapshot,d.dispatched_at,d.attempts`).Scan(&j.id, &j.channel, &j.snapshot, &j.started, &j.attempts)
	if e == nil {
		if e = s.sendPostiz(ctx, j); e != nil {
			return e
		}
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	return s.reconcilePostiz(ctx)
}

func (s *Server) sendPostiz(ctx context.Context, j postizJob) error {
	channels, e := s.loadPostizChannels(ctx, false)
	if e != nil {
		return s.postizSendError(ctx, j, e, false)
	}
	enabled := false
	for _, channel := range channels {
		if channel.ID == j.channel && channel.Supported && channel.Provider == str(j.snapshot["provider"]) {
			enabled = true
			break
		}
	}
	if !enabled {
		return s.finishPostiz(ctx, j, "failed", "", "Selected Postiz account is disconnected, disabled or changed provider", 0)
	}
	attachments := []Doc{}
	if source := str(j.snapshot["mediaUrl"]); source != "" {
		var uploaded struct{ ID, Path string }
		e := s.postiz.request(ctx, http.MethodPost, "/upload-from-url", Doc{"url": source}, &uploaded)
		if e != nil {
			return s.postizSendError(ctx, j, e, false)
		}
		if uploaded.ID == "" || uploaded.Path == "" {
			return s.postizSendError(ctx, j, errors.New("Postiz returned an incomplete media upload"), false)
		}
		attachments = append(attachments, Doc{"id": uploaded.ID, "path": uploaded.Path})
	}
	// Recheck eligibility after upload and before submitting the post. A
	// withdrawal concurrent with POST leaves the outcome visible for review.
	var active bool
	e = s.Store.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM creata_postiz_delivery d JOIN creata_records p ON p.kind='publication'
		AND p.id=d.article_id AND (p.data->>'revision')::int=d.revision WHERE d.id=$1 AND d.status='sending')`, j.id).Scan(&active)
	if e != nil {
		return e
	}
	if !active {
		return s.finishPostiz(ctx, j, "cancelled", "", "", 0)
	}
	post := Doc{"integration": Doc{"id": j.channel}, "value": []Doc{{"content": j.snapshot["content"], "image": attachments}}, "settings": j.snapshot["settings"]}
	request := Doc{"type": j.snapshot["mode"], "date": j.started.UTC().Format(time.RFC3339Nano), "shortLink": false, "tags": []any{}, "posts": []Doc{post}}
	var created []struct{ PostID, Integration string }
	if e = s.postiz.request(ctx, http.MethodPost, "/posts", request, &created); e != nil {
		return s.postizSendError(ctx, j, e, true)
	}
	if len(created) != 1 || created[0].PostID == "" || created[0].Integration != j.channel {
		return s.finishPostiz(ctx, j, "uncertain", "", "Postiz did not confirm the selected account and post ID", 2*time.Second)
	}
	status := "accepted"
	if j.snapshot["mode"] == "draft" {
		status = "draft"
	}
	return s.finishPostiz(ctx, j, status, created[0].PostID, "", 2*time.Second)
}

func (s *Server) postizSendError(ctx context.Context, j postizJob, e error, creating bool) error {
	var remote *postizError
	status, delay := "uncertain", 2*time.Second
	isHTTP := errors.As(e, &remote)
	switch {
	case isHTTP && remote.status == 429:
		status, delay = "queued", remote.retry
		if err := s.throttlePostiz(ctx, delay); err != nil {
			return err
		}
	case isHTTP && remote.status >= 400 && remote.status < 500 && remote.status != 408:
		status = "failed"
	case !creating || errors.Is(e, errPostizNotConnected):
		// Neither a failed upload nor a connection that never opened can
		// have created a social post.
		status, delay = "queued", time.Minute*time.Duration(j.attempts)
	}
	if status == "queued" && j.attempts >= 8 && (!isHTTP || remote.status != 429) {
		status = "failed"
	}
	return s.finishPostiz(ctx, j, status, "", e.Error(), delay)
}

func (s *Server) throttlePostiz(ctx context.Context, delay time.Duration) error {
	_, e := s.Store.Pool.Exec(ctx, `INSERT INTO creata_postiz_throttle(scope,until_at) VALUES('worker',$1)
		ON CONFLICT(scope) DO UPDATE SET until_at=greatest(creata_postiz_throttle.until_at,excluded.until_at)`, time.Now().Add(delay))
	return e
}

func (s *Server) finishPostiz(ctx context.Context, j postizJob, status, postID, message string, delay time.Duration) error {
	// If withdrawal raced with a 429, cancel its retry. This same statement
	// never converts a possibly accepted request into a sendable job.
	_, e := s.Store.Pool.Exec(ctx, `UPDATE creata_postiz_delivery d SET status=CASE WHEN $2='queued'
		AND NOT EXISTS(SELECT 1 FROM creata_records p WHERE p.kind='publication' AND p.id=d.article_id AND (p.data->>'revision')::int=d.revision)
		THEN 'cancelled' ELSE $2 END, post_id=$3,error=$4,next_at=$5,updated_at=now()
		WHERE d.id=$1 AND d.status='sending' AND d.attempts=$6`, j.id, status, postID, message, time.Now().Add(delay), j.attempts)
	return e
}

// Batch status reads by dispatch day so a calendar fetch reconciles multiple
// channels. The lease is persisted first to coordinate multiple API replicas.
func (s *Server) reconcilePostiz(ctx context.Context) error {
	rows, e := s.Store.Pool.Query(ctx, `WITH first_due AS (
		SELECT dispatched_at AS first_at FROM creata_postiz_delivery WHERE status IN ('uncertain','accepted','draft') AND next_at<=now()
		AND NOT EXISTS(SELECT 1 FROM creata_postiz_throttle WHERE scope='worker' AND until_at>now())
		ORDER BY next_at LIMIT 1), batch AS (
		SELECT d.id FROM creata_postiz_delivery d,first_due f WHERE d.status IN ('uncertain','accepted','draft') AND d.next_at<=now()
		AND d.dispatched_at BETWEEN f.first_at-interval '12 hours' AND f.first_at+interval '12 hours' LIMIT 200 FOR UPDATE OF d SKIP LOCKED)
		UPDATE creata_postiz_delivery d SET next_at=now()+interval '10 minutes' FROM batch b WHERE d.id=b.id
		RETURNING d.id,d.channel_id,d.post_id,d.status,d.snapshot,d.dispatched_at`)
	if e != nil {
		return e
	}
	jobs := []postizJob{}
	for rows.Next() {
		var j postizJob
		if e = rows.Scan(&j.id, &j.channel, &j.postID, &j.status, &j.snapshot, &j.started); e != nil {
			rows.Close()
			return e
		}
		jobs = append(jobs, j)
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return e
	}
	if len(jobs) == 0 {
		return nil
	}
	start, end := jobs[0].started, jobs[0].started
	for _, j := range jobs {
		if j.started.Before(start) {
			start = j.started
		}
		if j.started.After(end) {
			end = j.started
		}
	}
	query := url.Values{"startDate": {start.Add(-24 * time.Hour).UTC().Format(time.RFC3339)}, "endDate": {end.Add(24 * time.Hour).UTC().Format(time.RFC3339)}}
	var calendar struct{ Posts []postizRemote }
	if e = s.postiz.request(ctx, http.MethodGet, "/posts?"+query.Encode(), nil, &calendar); e != nil {
		var remote *postizError
		if errors.As(e, &remote) && remote.status == 429 {
			return s.throttlePostiz(ctx, remote.retry)
		}
		return e
	}
	for _, j := range jobs {
		var found *postizRemote
		for i := range calendar.Posts {
			p := &calendar.Posts[i]
			if p.ID == "" || p.Integration.ID != j.channel {
				continue
			}
			if j.postID != "" && p.ID != j.postID {
				continue
			}
			if j.postID == "" && strings.TrimSpace(p.Content) != str(j.snapshot["content"]) {
				continue
			}
			if found != nil {
				found = nil
				break
			}
			found = p
		}
		status, postID, link, message, delay := j.status, j.postID, "", "", time.Hour
		if j.status == "uncertain" {
			message = "Awaiting confirmation in the Postiz calendar"
		}
		if j.status == "draft" {
			delay = 24 * time.Hour
		}
		if found != nil {
			switch found.State {
			case "DRAFT":
				status, delay = "draft", 24*time.Hour
			case "QUEUE":
				status, delay = "accepted", 10*time.Minute
			case "PUBLISHED":
				status = "published"
			case "ERROR":
				status = "failed"
			default:
				continue
			}
			postID, message = found.ID, ""
			if status == "failed" {
				message = "Postiz reports that the platform rejected this post"
			}
			if u, e := url.Parse(found.ReleaseURL); e == nil && u.User == nil && u.Hostname() != "" && (u.Scheme == "https" || u.Scheme == "http") {
				link = u.String()
			}
		}
		_, e = s.Store.Pool.Exec(ctx, `UPDATE creata_postiz_delivery SET status=$2,post_id=$3,release_url=$4,error=$5,next_at=$6,updated_at=now() WHERE id=$1 AND status=$7`, j.id, status, postID, link, message, time.Now().Add(delay), j.status)
		if e != nil {
			return e
		}
	}
	return nil
}

type postizRemote struct {
	ID, Content, State, ReleaseURL string
	Integration                    struct{ ID string }
}
