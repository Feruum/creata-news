package news

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"
)

const launchSubmission = "launch_submission"

var launchSubmissionFields = []string{
	"name", "tagline", "description", "team", "url", "launchDate", "country", "categoryId", "photo",
}

func (s *Server) launchSubmissionRoutes() {
	s.handle("GET "+base+"/account/launch-submissions", s.listFounderSubmissions)
	s.handle("POST "+base+"/account/launch-submissions", s.saveLaunchSubmission)
	s.handle("PATCH "+base+"/account/launch-submissions/{id}", s.saveLaunchSubmission)
	s.handle("POST "+base+"/account/launch-submissions/{id}/submit", s.submitLaunch)
	s.handle("GET "+base+"/admin/launch-submissions", s.listLaunchReviewQueue)
	s.handle("POST "+base+"/admin/launch-submissions/{id}/review", s.reviewLaunchSubmission)
}

func submissionResponse(d Doc) Doc {
	out := clone(d)
	delete(out, "ownerId")
	return out
}

func (s *Server) requireLaunchModerator(r *http.Request) (Doc, error) {
	u, e := s.require(r, true)
	if e != nil {
		return nil, e
	}
	if !has(u["roles"], "editor") && !has(u["roles"], "chief_editor") && !has(u["roles"], "admin") {
		return nil, errStatus(403, "Launch moderation access denied")
	}
	return u, nil
}

func (s *Server) listLaunchSubmissions(w http.ResponseWriter, r *http.Request, field, value string) error {
	rows, e := s.Store.Pool.Query(r.Context(), `
		SELECT submission.data, COALESCE(launch.data->>'status' = 'published', false)
		FROM creata_records submission
		LEFT JOIN creata_records launch ON launch.kind = 'launch' AND launch.id = submission.id
		WHERE submission.kind = $1 AND submission.data->>($2::text) = $3
		ORDER BY submission.created_at DESC, submission.id`, launchSubmission, field, value)
	if e != nil {
		return e
	}
	defer rows.Close()
	visible := []Doc{}
	for rows.Next() {
		var raw []byte
		var published bool
		if e = rows.Scan(&raw, &published); e != nil {
			return e
		}
		var item Doc
		if e = json.Unmarshal(raw, &item); e != nil {
			return e
		}
		if str(item["status"]) == "published" && !published {
			item["status"] = "unpublished"
			delete(item, "launchSlug")
		}
		delete(item, "ownerId")
		visible = append(visible, item)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	respond(w, 200, Doc{"items": visible, "total": len(visible)})
	return nil
}

func (s *Server) listFounderSubmissions(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, false)
	if e != nil {
		return e
	}
	return s.listLaunchSubmissions(w, r, "ownerId", str(u["id"]))
}

func (s *Server) listLaunchReviewQueue(w http.ResponseWriter, r *http.Request) error {
	if _, e := s.requireLaunchModerator(r); e != nil {
		return e
	}
	return s.listLaunchSubmissions(w, r, "status", "in_review")
}

func editableLaunchSubmission(ctx context.Context, t *Tx, submissionID, ownerID string, input Doc) (Doc, error) {
	d, e := t.Get(ctx, launchSubmission, submissionID)
	if e != nil {
		return nil, e
	}
	if str(d["ownerId"]) != ownerID {
		return nil, errStatus(404, "Launch submission not found")
	}
	if number(input["expectedRevision"]) != number(d["revision"]) {
		return nil, errStatus(409, "Launch submission changed; reload before continuing")
	}
	if status := str(d["status"]); status != "draft" && status != "changes_requested" {
		return nil, errStatus(409, "Launch submission is not editable")
	}
	return d, nil
}

func saveSubmission(ctx context.Context, t *Tx, submission, actor Doc, action string) error {
	submission["revision"] = integer(submission["revision"]) + 1
	submission["updatedAt"] = now()
	if e := t.Put(ctx, launchSubmission, submission); e != nil {
		return e
	}
	return t.Audit(ctx, actor, "launch_submission."+action, str(submission["id"]), nil)
}

func (s *Server) saveLaunchSubmission(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, false)
	if e != nil {
		return e
	}
	input, e := decode(r)
	if e != nil {
		return e
	}
	ctx := r.Context()
	status := http.StatusCreated
	var out Doc
	e = s.Store.Write(ctx, func(t *Tx) error {
		if r.Method == "PATCH" {
			status = http.StatusOK
			out, e = editableLaunchSubmission(ctx, t, r.PathValue("id"), str(u["id"]), input)
			if e != nil {
				return e
			}
		} else {
			out = Doc{
				"id": id(), "ownerId": u["id"], "status": "draft", "createdAt": now(),
				"name": "", "tagline": localized(""), "description": localized(""), "team": localized(""),
				"url": "", "launchDate": "", "country": "", "categoryId": "", "photo": "",
			}
		}
		for _, field := range launchSubmissionFields {
			if value, ok := input[field]; ok {
				if field == "tagline" || field == "description" || field == "team" {
					text := object(value)
					value = Doc{"ru": text["ru"], "kk": text["kk"], "en": text["en"]}
				}
				out[field] = value
			}
		}
		if e = validateLaunchContent(out, false); e != nil {
			return e
		}
		return saveSubmission(ctx, t, out, u, "save")
	})
	if e != nil {
		return e
	}
	respond(w, status, submissionResponse(out))
	return nil
}

func (s *Server) submitLaunch(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, false)
	if e != nil {
		return e
	}
	input, e := decode(r)
	if e != nil {
		return e
	}
	ctx := r.Context()
	var out Doc
	e = s.Store.Write(ctx, func(t *Tx) error {
		out, e = editableLaunchSubmission(ctx, t, r.PathValue("id"), str(u["id"]), input)
		if e != nil {
			return e
		}
		if e = validateLaunchContent(out, true); e != nil {
			return e
		}
		if str(out["categoryId"]) != "" {
			if _, e = t.Get(ctx, "category", str(out["categoryId"])); e != nil {
				return errStatus(400, "Choose an existing category")
			}
		}
		out["status"] = "in_review"
		return saveSubmission(ctx, t, out, u, "submit")
	})
	if e != nil {
		return e
	}
	respond(w, 200, submissionResponse(out))
	return nil
}

func publishLaunchSubmission(ctx context.Context, t *Tx, submission Doc) error {
	submissionID := str(submission["id"])
	parts := strings.FieldsFunc(strings.ToLower(str(submission["name"])), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	prefix := strings.Join(parts, "-")
	if prefix == "" {
		prefix = "launch"
	}
	if len(prefix) > 67 {
		prefix = strings.TrimRight(prefix[:67], "-")
	}
	slug := prefix + "-" + submissionID
	launch := Doc{"id": submissionID, "createdAt": now(), "updatedAt": now()}
	input := Doc{"slug": slug, "status": "published"}
	for _, field := range launchSubmissionFields {
		input[field] = submission[field]
	}
	if e := prepareLaunch(ctx, t, launch, input, false); e != nil {
		return e
	}
	if e := t.Put(ctx, "launch", launch); e != nil {
		return e
	}
	submission["launchSlug"] = slug
	submission["status"] = "published"
	delete(submission, "reviewNote")
	return nil
}

func (s *Server) reviewLaunchSubmission(w http.ResponseWriter, r *http.Request) error {
	u, e := s.requireLaunchModerator(r)
	if e != nil {
		return e
	}
	input, e := decode(r)
	if e != nil {
		return e
	}
	action := str(input["action"])
	note := strings.TrimSpace(str(input["note"]))
	if action != "publish" && action != "request_changes" {
		return errStatus(400, "Unknown review action")
	}
	if action == "request_changes" && (note == "" || utf8.RuneCountInString(note) > 2000) {
		return errStatus(400, "Write a review note of up to 2000 characters")
	}
	ctx := r.Context()
	var out Doc
	e = s.Store.Write(ctx, func(t *Tx) error {
		out, e = t.Get(ctx, launchSubmission, r.PathValue("id"))
		if e != nil {
			return e
		}
		if number(input["expectedRevision"]) != number(out["revision"]) || str(out["status"]) != "in_review" {
			return errStatus(409, "Launch submission changed; reload the review queue")
		}
		if action == "publish" {
			if e = publishLaunchSubmission(ctx, t, out); e != nil {
				return e
			}
		} else {
			out["status"] = "changes_requested"
			out["reviewNote"] = note
		}
		return saveSubmission(ctx, t, out, u, action)
	})
	if e != nil {
		return e
	}
	respond(w, 200, submissionResponse(out))
	return nil
}
