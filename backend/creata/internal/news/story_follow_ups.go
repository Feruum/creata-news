package news

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) storyFollowUpRoutes() {
	s.handle("GET "+base+"/admin/story-follow-ups", s.storyFollowUps)
	s.handle("GET "+base+"/admin/articles/{id}/follow-up", s.getStoryFollowUp)
	s.handle("PUT "+base+"/admin/articles/{id}/follow-up", s.mutateStoryFollowUp)
	s.handle("DELETE "+base+"/admin/articles/{id}/follow-up", s.mutateStoryFollowUp)
}

func storyFollowUpItem(article, schedule Doc, assigneeName string) Doc {
	return Doc{
		"articleId": article["id"], "title": projectExtraText(article["title"]),
		"assigneeId": schedule["assigneeId"], "assigneeName": assigneeName,
		"dueDate": schedule["dueDate"], "updatedAt": schedule["updatedAt"],
	}
}

func (s *Server) storyFollowUps(w http.ResponseWriter, r *http.Request) error {
	u, err := s.require(r, true)
	if err != nil {
		return err
	}
	if !has(u["roles"], "editor") && !has(u["roles"], "chief_editor") {
		return errStatus(403, "Story follow-up access denied")
	}
	rows, err := s.Store.Pool.Query(r.Context(), `
		SELECT jsonb_build_object('id', article.id, 'title', article.data->'title',
			'layer', article.data->>'layer', 'categoryId', article.data->>'categoryId'),
			followup.data, COALESCE(assignee.data->>'name', '')
		FROM creata_records AS followup
		JOIN creata_records AS article ON article.kind='article' AND article.id=followup.id
		LEFT JOIN creata_records AS assignee ON assignee.kind='user' AND assignee.id=followup.data->>'assigneeId'
		WHERE followup.kind='story_follow_up'
		ORDER BY followup.data->>'dueDate', article.id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []Doc{}
	for rows.Next() {
		var article, schedule Doc
		var assigneeName string
		if err := rows.Scan(&article, &schedule, &assigneeName); err != nil {
			return err
		}
		if canPublish(u, article) {
			items = append(items, storyFollowUpItem(article, schedule, assigneeName))
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	respond(w, 200, Doc{"items": items, "total": len(items)})
	return nil
}

func (s *Server) getStoryFollowUp(w http.ResponseWriter, r *http.Request) error {
	u, err := s.require(r, true)
	if err != nil {
		return err
	}
	var article, schedule Doc
	var publishedUpdates []any
	var assigneeName string
	err = s.Store.Pool.QueryRow(r.Context(), `
		SELECT article.data, followup.data, publication.data->'storyUpdates', COALESCE(assignee.data->>'name', '')
		FROM creata_records AS article
		LEFT JOIN creata_records AS publication ON publication.kind='publication' AND publication.id=article.id
		LEFT JOIN creata_records AS followup ON followup.kind='story_follow_up' AND followup.id=article.id
		LEFT JOIN creata_records AS assignee ON assignee.kind='user' AND assignee.id=followup.data->>'assigneeId'
		WHERE article.kind='article' AND article.id=$1`, r.PathValue("id")).Scan(&article, &schedule, &publishedUpdates, &assigneeName)
	if errors.Is(err, pgx.ErrNoRows) {
		return errStatus(404, "Article not found")
	}
	if err != nil {
		return err
	}
	if !canPublish(u, article) {
		return errStatus(403, "Story follow-up access denied")
	}
	var item Doc
	if schedule != nil {
		item = storyFollowUpItem(article, schedule, assigneeName)
	}
	eligible := len(array(article["storyUpdates"])) > 0 || len(publishedUpdates) > 0
	respond(w, 200, Doc{"item": item, "eligible": eligible})
	return nil
}

func (s *Server) mutateStoryFollowUp(w http.ResponseWriter, r *http.Request) error {
	u, err := s.require(r, true)
	if err != nil {
		return err
	}
	var input Doc
	if r.Method == http.MethodPut {
		input, err = decode(r)
		if err != nil {
			return err
		}
	}
	ctx, articleID := r.Context(), r.PathValue("id")
	var item Doc
	err = s.Store.Write(ctx, func(t *Tx) error {
		article, err := t.Get(ctx, "article", articleID)
		if err != nil {
			return err
		}
		if !canPublish(u, article) {
			return errStatus(403, "Story follow-up access denied")
		}
		if r.Method == http.MethodDelete {
			return completeStoryFollowUp(ctx, t, u, articleID)
		}
		item, err = setStoryFollowUp(ctx, t, u, article, input)
		return err
	})
	if err != nil {
		return err
	}
	if r.Method == http.MethodDelete {
		respond(w, 200, Doc{"ok": true})
		return nil
	}
	respond(w, 200, Doc{"item": item})
	return nil
}

func setStoryFollowUp(ctx context.Context, t *Tx, u, article, input Doc) (Doc, error) {
	articleID, dueDate, assigneeID := str(article["id"]), str(input["dueDate"]), str(input["assigneeId"])
	date, err := time.Parse(time.DateOnly, dueDate)
	if err != nil || date.Year() < 1 || date.Format(time.DateOnly) != dueDate {
		return nil, errStatus(400, "dueDate must be a valid calendar date in YYYY-MM-DD format")
	}
	if len(input) != 2 || assigneeID == "" || strings.TrimSpace(assigneeID) != assigneeID || strings.ContainsRune(assigneeID, 0) {
		return nil, errStatus(400, "Choose an eligible editor as the assignee")
	}
	if len(array(article["storyUpdates"])) == 0 {
		var publishedUpdates []any
		err := t.QueryRow(ctx, `SELECT data->'storyUpdates' FROM creata_records
			WHERE kind='publication' AND id=$1`, articleID).Scan(&publishedUpdates)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if len(publishedUpdates) == 0 {
			return nil, errStatus(400, "Story follow-ups require a timeline update")
		}
	}
	assignee, err := t.Get(ctx, "user", assigneeID)
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.status == http.StatusNotFound {
		return nil, errStatus(400, "Choose an eligible editor as the assignee")
	}
	if err != nil {
		return nil, err
	}
	if !canPublish(assignee, article) {
		return nil, errStatus(400, "Choose an eligible editor as the assignee")
	}
	schedule := Doc{"id": articleID, "dueDate": dueDate, "assigneeId": assigneeID, "updatedAt": now()}
	if err := t.Put(ctx, "story_follow_up", schedule); err != nil {
		return nil, err
	}
	if err := t.Audit(ctx, u, "story_follow_up.set", articleID, Doc{"dueDate": dueDate, "assigneeId": assigneeID}); err != nil {
		return nil, err
	}
	return storyFollowUpItem(article, schedule, str(assignee["name"])), nil
}

func completeStoryFollowUp(ctx context.Context, t *Tx, u Doc, articleID string) error {
	var schedule Doc
	err := t.QueryRow(ctx, `DELETE FROM creata_records WHERE kind='story_follow_up' AND id=$1 RETURNING data`, articleID).Scan(&schedule)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return t.Audit(ctx, u, "story_follow_up.complete", articleID, Doc{"dueDate": schedule["dueDate"], "assigneeId": schedule["assigneeId"]})
}
