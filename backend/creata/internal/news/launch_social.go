package news

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (s *Server) launchSocialRoutes() {
	s.handle("GET "+base+"/account/launch-votes", s.launchVotes)
	s.handle("POST "+base+"/account/launches/{id}/vote", s.launchVote)
	s.handle("GET "+base+"/public/launches/{slug}/comments", s.launchComments)
	s.handle("POST "+base+"/account/launches/{id}/comments", s.launchComment)
	s.handle("DELETE "+base+"/account/launches/{id}/comments/{commentId}", s.deleteLaunchComment)
}

func (s *Server) launchStats(ctx context.Context, items []Doc) error {
	ids := []string{}
	byID := map[string]Doc{}
	for _, item := range items {
		item["votes"], item["commentsCount"] = 0, 0
		ids = append(ids, str(item["id"]))
		byID[str(item["id"])] = item
	}
	if len(ids) == 0 {
		return nil
	}
	rows, e := s.Store.Pool.Query(ctx, "SELECT data->>'launchId',kind,count(*) FROM creata_records WHERE kind IN ('launch_vote','launch_comment') AND data->>'launchId'=ANY($1::text[]) GROUP BY data->>'launchId',kind", ids)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var launchID, kind string
		var count int
		if e = rows.Scan(&launchID, &kind, &count); e != nil {
			return e
		}
		field := "votes"
		if kind == "launch_comment" {
			field = "commentsCount"
		}
		byID[launchID][field] = count
	}
	return rows.Err()
}

func (s *Server) launchVotes(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, false)
	if e != nil {
		return e
	}
	rows, e := s.Store.Pool.Query(r.Context(), "SELECT data->>'launchId' FROM creata_records WHERE kind='launch_vote' AND data->>'userId'=$1", str(u["id"]))
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var item string
		if e = rows.Scan(&item); e != nil {
			return e
		}
		items = append(items, item)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	respond(w, 200, Doc{"items": items, "total": len(items)})
	return nil
}

func (s *Server) launchVote(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, false)
	if e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	voted, ok := d["voted"].(bool)
	if !ok {
		return errStatus(400, "voted must be a boolean")
	}
	launchID := r.PathValue("id")
	count := 0
	e = s.Store.Write(r.Context(), func(tx *Tx) error {
		launch, e := tx.Get(r.Context(), "launch", launchID)
		if e != nil {
			return e
		}
		if str(launch["status"]) != "published" {
			return errStatus(404, "Startup not found")
		}
		voteID := hash(str(u["id"]) + ":" + launchID)
		if voted {
			e = tx.Put(r.Context(), "launch_vote", Doc{"id": voteID, "launchId": launchID, "userId": u["id"]})
		} else {
			e = tx.Delete(r.Context(), "launch_vote", voteID)
		}
		if e != nil {
			return e
		}
		return tx.QueryRow(r.Context(), "SELECT count(*) FROM creata_records WHERE kind='launch_vote' AND data->>'launchId'=$1", launchID).Scan(&count)
	})
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"voted": voted, "votes": count})
	return nil
}

func publicLaunchComment(d, user Doc) Doc {
	canDelete := user != nil && (str(user["id"]) == str(d["authorId"]) || has(user["roles"], "admin") || has(user["roles"], "chief_editor") || has(user["roles"], "editor"))
	return Doc{"id": d["id"], "launchId": d["launchId"], "authorName": d["authorName"], "body": d["body"], "createdAt": d["createdAt"], "canDelete": canDelete}
}

func (s *Server) launchComments(w http.ResponseWriter, r *http.Request) error {
	launches, e := s.Store.List(r.Context(), "launch")
	if e != nil {
		return e
	}
	launchID := ""
	for _, launch := range launches {
		if str(launch["slug"]) == r.PathValue("slug") && str(launch["status"]) == "published" {
			launchID = str(launch["id"])
			break
		}
	}
	if launchID == "" {
		return errStatus(404, "Startup not found")
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, e = strconv.Atoi(raw)
		if e != nil || offset < 0 || offset > 10000 {
			return errStatus(400, "Invalid comment offset")
		}
	}
	var total int
	if e = s.Store.Pool.QueryRow(r.Context(), "SELECT count(*) FROM creata_records WHERE kind='launch_comment' AND data->>'launchId'=$1", launchID).Scan(&total); e != nil {
		return e
	}
	u, e := s.user(r)
	if e != nil {
		return e
	}
	rows, e := s.Store.Pool.Query(r.Context(), "SELECT data FROM creata_records WHERE kind='launch_comment' AND data->>'launchId'=$1 ORDER BY created_at DESC,id DESC LIMIT 20 OFFSET $2", launchID, offset)
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []Doc{}
	for rows.Next() {
		var d Doc
		if e = rows.Scan(&d); e != nil {
			return e
		}
		items = append(items, publicLaunchComment(d, u))
	}
	if e = rows.Err(); e != nil {
		return e
	}
	respond(w, 200, Doc{"items": items, "total": total})
	return nil
}

func (s *Server) launchComment(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, false)
	if e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	body := strings.TrimSpace(str(d["body"]))
	if utf8.RuneCountInString(body) < 1 || utf8.RuneCountInString(body) > 2000 {
		return errStatus(400, "Comment must contain 1–2000 characters")
	}
	clientID := str(d["clientId"])
	if !launchSlug.MatchString(clientID) || len(clientID) > 80 {
		return errStatus(400, "A valid comment request ID is required")
	}
	launchID := r.PathValue("id")
	commentID := hash(str(u["id"]) + ":" + launchID + ":" + clientID)
	var out Doc
	e = s.Store.Write(r.Context(), func(tx *Tx) error {
		launch, e := tx.Get(r.Context(), "launch", launchID)
		if e != nil {
			return e
		}
		if str(launch["status"]) != "published" {
			return errStatus(404, "Startup not found")
		}
		if existing, e := tx.Get(r.Context(), "launch_comment", commentID); e == nil {
			out = existing
			return nil
		} else {
			var api *apiError
			if !errors.As(e, &api) || api.status != 404 {
				return e
			}
		}
		out = Doc{"id": commentID, "launchId": launchID, "authorId": u["id"], "authorName": u["name"], "body": body, "createdAt": now()}
		return tx.Put(r.Context(), "launch_comment", out)
	})
	if e != nil {
		return e
	}
	respond(w, 200, publicLaunchComment(out, u))
	return nil
}

func (s *Server) deleteLaunchComment(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, false)
	if e != nil {
		return e
	}
	e = s.Store.Write(r.Context(), func(tx *Tx) error {
		comment, e := tx.Get(r.Context(), "launch_comment", r.PathValue("commentId"))
		if e != nil {
			return e
		}
		if str(comment["launchId"]) != r.PathValue("id") {
			return errStatus(404, "Comment not found")
		}
		if !boolean(publicLaunchComment(comment, u)["canDelete"]) {
			return errStatus(403, "Comment removal denied")
		}
		if e = tx.Delete(r.Context(), "launch_comment", str(comment["id"])); e != nil {
			return e
		}
		return tx.Audit(r.Context(), u, "launch_comment.delete", str(comment["id"]), nil)
	})
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"ok": true})
	return nil
}
