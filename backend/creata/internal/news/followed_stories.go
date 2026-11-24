package news

import (
	"net/http"
	"slices"
	"sort"
)

func (s *Server) followedStoryRoutes() {
	s.handle("GET "+base+"/account/followed-stories", s.followedStories)
	s.handle("POST "+base+"/account/followed-stories/{id}", s.followedStoryMutation)
	s.handle("DELETE "+base+"/account/followed-stories/{id}", s.followedStoryMutation)
	s.handle("POST "+base+"/account/followed-stories/{id}/read", s.readFollowedStory)
}

func (s *Server) followedStories(w http.ResponseWriter, r *http.Request) error {
	u, err := s.require(r, false)
	if err != nil {
		return err
	}
	ctx := r.Context()
	rows, err := s.Store.Pool.Query(ctx, `
		SELECT publication.data, follow.data->'seenUpdateIds' FROM creata_records AS follow
		JOIN creata_records AS publication ON publication.kind='publication' AND publication.id=follow.data->>'articleId'
		WHERE follow.kind='story_follow' AND follow.data->>'userId'=$1`, str(u["id"]))
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []Doc{}
	unreadByStory := map[string][]string{}
	for rows.Next() {
		var article Doc
		var seenIDs []string
		if err := rows.Scan(&article, &seenIDs); err != nil {
			return err
		}
		if len(array(article["storyUpdates"])) == 0 {
			continue
		}
		public := publicArticle(article)
		items = append(items, public)
		unreadByStory[str(public["id"])] = unseenStoryUpdateIDs(public, seenIDs)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	sort.Slice(items, func(i, j int) bool {
		left := parseTime(object(array(items[i]["storyUpdates"])[0])["occurredAt"])
		right := parseTime(object(array(items[j]["storyUpdates"])[0])["occurredAt"])
		if left.Equal(right) {
			return str(items[i]["id"]) < str(items[j]["id"])
		}
		return left.After(right)
	})
	for i, article := range items {
		items[i] = s.decorate(ctx, article)
	}
	respond(w, 200, Doc{"items": items, "unreadUpdateIds": unreadByStory})
	return nil
}

func (s *Server) followedStoryMutation(w http.ResponseWriter, r *http.Request) error {
	u, err := s.require(r, false)
	if err != nil {
		return err
	}
	ctx := r.Context()
	userID, articleID := str(u["id"]), r.PathValue("id")
	key := hash(userID + "\x00" + articleID)
	err = s.Store.Write(ctx, func(t *Tx) error {
		if r.Method == http.MethodDelete {
			return t.Delete(ctx, "story_follow", key)
		}
		publication, err := t.Get(ctx, "publication", articleID)
		if err != nil {
			return err
		}
		if len(array(publication["storyUpdates"])) == 0 {
			return errStatus(404, "Story not found")
		}
		_, err = t.Exec(ctx, `INSERT INTO creata_records(kind, id, data)
			VALUES('story_follow', $1, jsonb_build_object('userId',$2::text,'articleId',$3::text,'seenUpdateIds',$4::jsonb))
			ON CONFLICT(kind, id) DO NOTHING`, key, userID, articleID, unseenStoryUpdateIDs(publication, nil))
		return err
	})
	if err != nil {
		return err
	}
	respond(w, 200, Doc{"ok": true})
	return nil
}

func unseenStoryUpdateIDs(article Doc, seenIDs []string) []string {
	ids := []string{}
	for _, raw := range array(article["storyUpdates"]) {
		updateID := str(object(raw)["id"])
		if !slices.Contains(seenIDs, updateID) {
			ids = append(ids, updateID)
		}
	}
	return ids
}

func (s *Server) readFollowedStory(w http.ResponseWriter, r *http.Request) error {
	u, err := s.require(r, false)
	if err != nil {
		return err
	}
	d, err := decode(r)
	if err != nil {
		return err
	}
	updateIDs, ok := d["updateIds"].([]any)
	if !ok || len(updateIDs) > 40 || len(d) != 1 {
		return errStatus(400, "updateIds must be an array of up to 40 update IDs")
	}
	requested := make(map[string]bool, len(updateIDs))
	for _, raw := range updateIDs {
		updateID := str(raw)
		if !storyUpdateID.MatchString(updateID) {
			return errStatus(400, "Invalid story update ID")
		}
		requested[updateID] = true
	}
	ctx := r.Context()
	userID, articleID := str(u["id"]), r.PathValue("id")
	key := hash(userID + "\x00" + articleID)
	err = s.Store.Write(ctx, func(t *Tx) error {
		follow, err := t.Get(ctx, "story_follow", key)
		if err != nil {
			return err
		}
		if str(follow["userId"]) != userID || str(follow["articleId"]) != articleID {
			return errStatus(404, "Story follow not found")
		}
		publication, err := t.Get(ctx, "publication", articleID)
		if err != nil {
			return err
		}
		if len(array(publication["storyUpdates"])) == 0 {
			return errStatus(404, "Story not found")
		}
		seenIDs := []string{}
		for _, raw := range array(follow["seenUpdateIds"]) {
			seenIDs = append(seenIDs, str(raw))
		}
		previousCount := len(seenIDs)
		for _, updateID := range unseenStoryUpdateIDs(publication, seenIDs) {
			if requested[updateID] {
				seenIDs = append(seenIDs, updateID)
			}
		}
		if len(seenIDs) == previousCount {
			return nil
		}
		follow["id"] = key
		follow["seenUpdateIds"] = seenIDs
		return t.Put(ctx, "story_follow", follow)
	})
	if err != nil {
		return err
	}
	respond(w, 200, Doc{"ok": true})
	return nil
}
