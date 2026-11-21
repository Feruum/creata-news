package news

import (
	"context"
	"net/http"
	"strings"
	"time"
)

func (s *Server) publicRoutes() {
	s.handle("GET "+base+"/public/popular", s.popularPublic)
	s.handle("GET "+base+"/public/articles", s.articlesPublic)
	s.handle("GET "+base+"/public/articles/{slug}", s.articlePublic)
	for path, kind := range map[string]string{"categories": "category", "authors": "author", "opportunities": "opportunity"} {
		p, k := path, kind
		s.handle("GET "+base+"/public/"+p, func(w http.ResponseWriter, r *http.Request) error {
			if k == "opportunity" {
				return s.opportunitiesPublic(w, r)
			}
			return s.listResponse(w, r, k)
		})
		if k != "category" {
			s.handle("GET "+base+"/public/"+p+"/{id}", func(w http.ResponseWriter, r *http.Request) error {
				d, e := s.Store.Get(r.Context(), k, r.PathValue("id"))
				if e != nil {
					return e
				}
				if k == "opportunity" && str(d["status"]) != "approved" {
					return errStatus(404, "Opportunity not found")
				}
				respond(w, 200, d)
				return nil
			})
		}
	}
	s.handle("POST "+base+"/public/events", s.event)
}
func (s *Server) decorate(ctx context.Context, a Doc) Doc {
	c := clone(a)
	if au, e := s.Store.Get(ctx, "author", str(a["authorId"])); e == nil {
		c["author"] = au
	}
	return c
}
func (s *Server) articlesPublic(w http.ResponseWriter, r *http.Request) error {
	items, e := s.Store.List(r.Context(), "publication")
	if e != nil {
		return e
	}
	out := []Doc{}
	q := r.URL.Query()
	for _, a := range items {
		if q.Get("format") != "" && materialFormat(a) != q.Get("format") {
			continue
		}
		if q.Get("layer") != "" && str(a["layer"]) != q.Get("layer") {
			continue
		}
		if q.Get("category") != "" && str(a["categoryId"]) != q.Get("category") {
			continue
		}
		if q.Get("authorId") != "" && str(a["authorId"]) != q.Get("authorId") {
			continue
		}
		if q.Get("region") != "" && str(a["region"]) != q.Get("region") {
			continue
		}
		if search := strings.ToLower(q.Get("q")); search != "" {
			text := strings.ToLower(localize(a["title"], locale(q.Get("locale"))) + " " + localize(a["body"], locale(q.Get("locale"))))
			if !strings.Contains(text, search) {
				continue
			}
		}
		out = append(out, s.decorate(r.Context(), publicArticle(a)))
	}
	respond(w, 200, Doc{"items": out, "total": len(out)})
	return nil
}
func (s *Server) articlePublic(w http.ResponseWriter, r *http.Request) error {
	items, e := s.Store.List(r.Context(), "publication")
	if e != nil {
		return e
	}
	for _, a := range items {
		if str(a["slug"]) == r.PathValue("slug") {
			p := s.decorate(r.Context(), publicArticle(a))
			history, _ := s.Store.List(r.Context(), "publication_version")
			corrections := []Doc{}
			for _, v := range history {
				if str(v["articleId"]) == str(a["id"]) && str(v["note"]) != "" {
					corrections = append(corrections, Doc{"revision": v["revision"], "note": v["note"], "publishedAt": v["publishedAt"]})
				}
			}
			p["corrections"] = corrections
			respond(w, 200, p)
			return nil
		}
	}
	return errStatus(404, "Article not found")
}
func (s *Server) opportunitiesPublic(w http.ResponseWriter, r *http.Request) error {
	items, e := s.Store.List(r.Context(), "opportunity")
	if e != nil {
		return e
	}
	out := []Doc{}
	q := r.URL.Query()
	for _, o := range items {
		if str(o["status"]) != "approved" || !parseTime(o["deadline"]).After(time.Now()) {
			continue
		}
		if q.Get("type") != "" && str(o["type"]) != q.Get("type") {
			continue
		}
		if q.Get("category") != "" && str(o["categoryId"]) != q.Get("category") {
			continue
		}
		if search := strings.ToLower(q.Get("q")); search != "" && !strings.Contains(strings.ToLower(localize(o["title"], locale(q.Get("locale")))+" "+str(o["organization"])), search) {
			continue
		}
		out = append(out, o)
	}
	respond(w, 200, Doc{"items": out, "total": len(out)})
	return nil
}
func (s *Server) event(w http.ResponseWriter, r *http.Request) error {
	d, e := decode(r)
	if e != nil {
		return e
	}
	kind := str(d["type"])
	if !has([]string{"view", "save", "opportunity_click"}, kind) || str(d["targetId"]) == "" {
		return errStatus(400, "Invalid analytics event")
	}
	targetKind := "publication"
	if kind == "opportunity_click" {
		targetKind = "opportunity"
	}
	if _, e = s.Store.Get(r.Context(), targetKind, str(d["targetId"])); e != nil {
		return errStatus(404, "Public target not found")
	}
	eventID := str(d["eventId"])
	if eventID == "" {
		eventID = id()
	}
	u, _ := s.user(r)
	uid := ""
	if u != nil {
		uid = str(u["id"])
	}
	_, e = s.Store.Pool.Exec(r.Context(), "INSERT INTO creata_events(id,kind,target_id,user_id) VALUES($1,$2,$3,NULLIF($4,'')) ON CONFLICT DO NOTHING", hash(kind+":"+eventID), kind, d["targetId"], uid)
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"ok": true})
	return nil
}
func (s *Server) account(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, false)
	if e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	var out Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		out, e = t.Get(r.Context(), "user", str(u["id"]))
		if e != nil {
			return e
		}
		if e = updatePreferences(out, d); e != nil {
			return e
		}
		return t.Put(r.Context(), "user", out)
	})
	if e != nil {
		return e
	}
	respond(w, 200, safeUser(out))
	return nil
}
func updatePreferences(u, d Doc) error {
	for _, k := range []string{"locale", "stage", "topics", "frequency", "paused", "name"} {
		if v, ok := d[k]; ok {
			switch k {
			case "paused":
				if _, ok := v.(bool); !ok {
					return errStatus(400, "paused must be a boolean")
				}
			case "locale":
				if !has([]string{"ru", "kk", "en"}, str(v)) {
					return errStatus(400, "Unsupported locale")
				}
			case "stage":
				if !has([]string{"exploring", "trying", "building"}, str(v)) {
					return errStatus(400, "Invalid stage")
				}
			case "frequency":
				if !has([]string{"daily", "weekly"}, str(v)) {
					return errStatus(400, "Invalid frequency")
				}
			case "topics":
				if len(array(v)) > 12 {
					return errStatus(400, "Choose up to 12 topics")
				}
			case "name":
				if len(str(v)) > 200 {
					return errStatus(400, "Name is too long")
				}
			}
			u[k] = v
		}
	}
	u["updatedAt"] = now()
	return nil
}
func (s *Server) savedList(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, false)
	if e != nil {
		return e
	}
	items, e := s.Store.List(r.Context(), "save")
	if e != nil {
		return e
	}
	arts, opps := []Doc{}, []Doc{}
	for _, v := range items {
		if str(v["userId"]) != str(u["id"]) {
			continue
		}
		kind := "publication"
		if str(v["kind"]) == "opportunity" {
			kind = "opportunity"
		}
		d, e := s.Store.Get(r.Context(), kind, str(v["targetId"]))
		if e != nil {
			continue
		}
		if kind == "publication" {
			arts = append(arts, s.decorate(r.Context(), publicArticle(d)))
		} else if str(d["status"]) == "approved" {
			opps = append(opps, d)
		}
	}
	respond(w, 200, Doc{"articles": arts, "opportunities": opps})
	return nil
}
func (s *Server) savedMutation(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, false)
	if e != nil {
		return e
	}
	kind := r.PathValue("kind")
	if kind != "article" && kind != "opportunity" {
		return errStatus(400, "Invalid saved type")
	}
	recordKind := "publication"
	if kind == "opportunity" {
		recordKind = kind
	}
	if _, e = s.Store.Get(r.Context(), recordKind, r.PathValue("id")); e != nil {
		return e
	}
	key := str(u["id"]) + ":" + kind + ":" + r.PathValue("id")
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		if r.Method == "DELETE" {
			return t.Delete(r.Context(), "save", key)
		}
		return t.Put(r.Context(), "save", Doc{"id": key, "userId": u["id"], "kind": kind, "targetId": r.PathValue("id"), "createdAt": now()})
	})
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"ok": true})
	return nil
}
func (s *Server) telegramLink(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, false)
	if e != nil {
		return e
	}
	if s.Config.BotUsername == "" {
		return errStatus(503, "Telegram bot is not configured")
	}
	token := id() + id()
	expires := time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		return t.Put(r.Context(), "telegram_link", Doc{"id": hash(token), "userId": u["id"], "expiresAt": expires})
	})
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"url": "https://t.me/" + s.Config.BotUsername + "?start=link_" + token, "expiresAt": expires})
	return nil
}
