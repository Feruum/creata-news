package news

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (s *Server) digestRoutes() {
	s.handle("GET "+base+"/admin/digests", func(w http.ResponseWriter, r *http.Request) error {
		if _, e := s.require(r, true); e != nil {
			return e
		}
		return s.listResponse(w, r, "digest")
	})
	s.handle("POST "+base+"/admin/digests/preview", s.digestPreview)
	s.handle("POST "+base+"/admin/digests/{id}/{action}", s.digestAction)
	s.handle("POST "+base+"/internal/bot/users/upsert", s.botUpsert)
	s.handle("PATCH "+base+"/internal/bot/users/{telegramId}", s.botPreferences)
	s.handle("POST "+base+"/internal/bot/link", s.botLink)
	s.handle("GET "+base+"/internal/bot/topics", func(w http.ResponseWriter, r *http.Request) error {
		if e := s.service(r, "bot"); e != nil {
			return e
		}
		return s.listResponse(w, r, "category")
	})
	s.handle("POST "+base+"/internal/bot/digest", s.botDigest)
	s.handle("POST "+base+"/internal/bot/digests/{id}/{action}", s.manualDeliveryResult)
	s.handle("GET "+base+"/admin/deliveries", s.deliveryList)
	s.handle("POST "+base+"/admin/deliveries/{id}/resolve", s.deliveryResolve)
	s.handle("POST "+base+"/internal/deliveries/claim", s.claimDelivery)
	s.handle("POST "+base+"/internal/deliveries/{id}/{action}", s.finishDelivery)
}
func (s *Server) compose(t *Tx, ctx context.Context, u Doc, kind string) (Doc, error) {
	all, e := t.List(ctx, "publication")
	if e != nil {
		return nil, e
	}
	sort.SliceStable(all, func(i, j int) bool { return parseTime(all[i]["publishedAt"]).After(parseTime(all[j]["publishedAt"])) })
	lang := locale(str(u["locale"]))
	sent := map[string]bool{}
	if kind == "personal" {
		sentRecords, e := t.List(ctx, "sent_item")
		if e != nil {
			return nil, e
		}
		for _, v := range sentRecords {
			if str(v["userId"]) == str(u["id"]) {
				sent[str(v["articleId"])] = true
			}
		}
		pending, e := t.List(ctx, "delivery")
		if e != nil {
			return nil, e
		}
		for _, delivery := range pending {
			if str(delivery["userId"]) != str(u["id"]) || !has([]string{"sending", "unknown", "queued"}, str(delivery["status"])) {
				continue
			}
			digest, e := t.Get(ctx, "digest", str(delivery["digestId"]))
			if e != nil {
				continue
			}
			for _, v := range append(array(digest["articles"]), digest["outsideInterest"]) {
				if aid := str(object(v)["id"]); aid != "" {
					sent[aid] = true
				}
			}
		}
	}
	matches := func(a Doc) bool { return len(array(u["topics"])) == 0 || has(u["topics"], str(a["categoryId"])) }
	articles := []any{}
	chosen := map[string]bool{}
	for _, a := range all {
		if len(articles) >= 5 {
			break
		}
		if sent[str(a["id"])] || !matches(a) {
			continue
		}
		articles = append(articles, publicArticle(a))
		chosen[str(a["id"])] = true
	}
	var outside Doc
	for _, a := range all {
		if sent[str(a["id"])] || chosen[str(a["id"])] {
			continue
		}
		if len(array(u["topics"])) > 0 && !matches(a) {
			outside = publicArticle(a)
			break
		}
	}
	opps, e := t.List(ctx, "opportunity")
	if e != nil {
		return nil, e
	}
	sort.SliceStable(opps, func(i, j int) bool { return parseTime(opps[i]["deadline"]).Before(parseTime(opps[j]["deadline"])) })
	selected := []any{}
	for _, o := range opps {
		if len(selected) >= 3 {
			break
		}
		if str(o["status"]) != "approved" || !parseTime(o["deadline"]).After(time.Now()) {
			continue
		}
		if len(array(u["topics"])) > 0 && str(o["categoryId"]) != "" && !has(u["topics"], str(o["categoryId"])) {
			continue
		}
		selected = append(selected, o)
	}
	d := Doc{"id": id(), "locale": lang, "kind": kind, "status": "draft", "articles": articles, "opportunities": selected, "createdAt": now(), "userId": u["id"]}
	if outside != nil {
		d["outsideInterest"] = outside
	}
	d["text"] = s.digestText(d)
	return d, nil
}
func (s *Server) digestText(d Doc) string {
	lang := locale(str(d["locale"]))
	heading := map[string]string{"ru": "CR≡ATA News · Дайджест", "kk": "CR≡ATA News · Дайджест", "en": "CR≡ATA News · Digest"}[lang]
	var b strings.Builder
	b.WriteString(heading + "\n\n")
	for _, raw := range array(d["articles"]) {
		a := object(raw)
		b.WriteString("• " + localize(a["title"], lang) + "\n" + s.articleLink(a, lang) + "\n\n")
	}
	if a := object(d["outsideInterest"]); len(a) > 0 {
		label := map[string]string{"ru": "Вне ваших тем", "kk": "Жаңа тақырып", "en": "Beyond your interests"}[lang]
		b.WriteString(label + ": " + localize(a["title"], lang) + "\n" + s.articleLink(a, lang) + "\n\n")
	}
	for _, raw := range array(d["opportunities"]) {
		o := object(raw)
		b.WriteString("→ " + localize(o["title"], lang) + "\n" + str(o["url"]) + "\n\n")
	}
	if len(array(d["articles"])) == 0 && len(array(d["opportunities"])) == 0 && len(object(d["outsideInterest"])) == 0 {
		b.WriteString(map[string]string{"ru": "Пока нет новых опубликованных материалов для подборки.", "kk": "Әзірге жинаққа арналған жаңа жарияланымдар жоқ.", "en": "There are no new published stories for this digest yet."}[lang])
	}
	return b.String()
}
func (s *Server) articleLink(a Doc, lang string) string {
	return strings.TrimRight(s.Config.PublicURL, "/") + "/" + lang + "/articles/" + str(a["slug"])
}
func (s *Server) digestPreview(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !has(u["roles"], "chief_editor") && !has(u["roles"], "editor") {
		return errStatus(403, "Digest editing denied")
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	var out Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		out, e = s.compose(t, r.Context(), Doc{"id": "channel", "locale": locale(str(d["locale"])), "topics": []any{}}, "channel")
		if e != nil {
			return e
		}
		return t.Put(r.Context(), "digest", out)
	})
	if e != nil {
		return e
	}
	respond(w, 201, out)
	return nil
}
func (s *Server) digestAction(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !has(u["roles"], "chief_editor") && !has(u["roles"], "editor") {
		return errStatus(403, "Digest publishing denied")
	}
	var d Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		d, e = t.Get(r.Context(), "digest", r.PathValue("id"))
		if e != nil {
			return e
		}
		if str(d["kind"]) != "channel" {
			return errStatus(400, "Only channel editions are managed here")
		}
		switch r.PathValue("action") {
		case "approve":
			if str(d["status"]) != "draft" {
				return errStatus(409, "Only draft digests can be approved")
			}
			d["status"] = "approved"
			d["approvedBy"] = u["id"]
		case "send":
			if str(d["status"]) == "queued" || str(d["status"]) == "sent" {
				return nil
			}
			if str(d["status"]) != "approved" {
				return errStatus(409, "Digest must be approved before sending")
			}
			if s.Config.ChannelID == "" {
				return errStatus(503, "Telegram channel is not configured")
			}
			if e = s.enqueueDelivery(t, r.Context(), d, s.Config.ChannelID, "channel"); e != nil {
				return e
			}
			d["status"] = "queued"
		default:
			return errStatus(400, "Unknown digest action")
		}
		if e = t.Put(r.Context(), "digest", d); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "digest."+r.PathValue("action"), str(d["id"]), nil)
	})
	if e != nil {
		return e
	}
	respond(w, 200, d)
	return nil
}
func findTelegram(t *Tx, ctx context.Context, tg string) (Doc, error) {
	var b []byte
	e := t.QueryRow(ctx, "SELECT data FROM creata_records WHERE kind='user' AND data->>'telegramId'=$1", tg).Scan(&b)
	if e != nil {
		return nil, e
	}
	var u Doc
	e = json.Unmarshal(b, &u)
	return u, e
}
func validTelegramID(v string) bool {
	if len(v) == 0 || len(v) > 20 {
		return false
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
func (s *Server) botUpsert(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "bot"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	tg := str(d["telegramId"])
	if !validTelegramID(tg) {
		return errStatus(400, "Invalid Telegram identifier")
	}
	var u Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		u, e = findTelegram(t, r.Context(), tg)
		if e == nil {
			return nil
		}
		name := str(d["username"])
		if name == "" {
			name = "Creata reader"
		}
		u = newUser(id(), name, locale(str(d["locale"])))
		u["telegramId"] = tg
		u["telegramLinked"] = true
		return t.Put(r.Context(), "user", u)
	})
	if e != nil {
		return e
	}
	respond(w, 200, safeUser(u))
	return nil
}
func (s *Server) botPreferences(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "bot"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	var u Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		u, e = findTelegram(t, r.Context(), r.PathValue("telegramId"))
		if e != nil {
			return errStatus(404, "Start the bot first")
		}
		if e = updatePreferences(u, d); e != nil {
			return e
		}
		return t.Put(r.Context(), "user", u)
	})
	if e != nil {
		return e
	}
	respond(w, 200, safeUser(u))
	return nil
}
func (s *Server) botLink(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "bot"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	if !validTelegramID(str(d["telegramId"])) {
		return errStatus(400, "Invalid Telegram identifier")
	}
	var u Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		link, e := t.Get(r.Context(), "telegram_link", hash(str(d["token"])))
		if e != nil || !parseTime(link["expiresAt"]).After(time.Now()) {
			return errStatus(400, "Link expired or already used")
		}
		u, e = t.Get(r.Context(), "user", str(link["userId"]))
		if e != nil {
			return e
		}
		tg := str(d["telegramId"])
		if current := str(u["telegramId"]); current != "" && current != tg {
			return errStatus(409, "This account already has another Telegram connection")
		}
		other, err := findTelegram(t, r.Context(), tg)
		if err == nil && str(other["id"]) != str(u["id"]) {
			identities, e := t.List(r.Context(), "identity")
			if e != nil {
				return e
			}
			for _, i := range identities {
				if str(i["userId"]) == str(other["id"]) {
					return errStatus(409, "Telegram belongs to another signed-in account")
				}
			}
			if len(array(other["roles"])) != 1 || !has(other["roles"], "reader") {
				return errStatus(409, "Telegram belongs to another staff account")
			}
			if e = t.Delete(r.Context(), "user", str(other["id"])); e != nil {
				return e
			}
			for _, kind := range []string{"delivery", "digest", "digest_period", "sent_item"} {
				records, e := t.List(r.Context(), kind)
				if e != nil {
					return e
				}
				for _, record := range records {
					if str(record["userId"]) != str(other["id"]) {
						continue
					}
					record["userId"] = u["id"]
					if kind == "sent_item" {
						oldID := str(record["id"])
						record["id"] = str(u["id"]) + ":" + str(record["articleId"])
						if e = t.Delete(r.Context(), kind, oldID); e != nil {
							return e
						}
					}
					if e = t.Put(r.Context(), kind, record); e != nil {
						return e
					}
				}
			}
		}
		u["telegramId"] = tg
		u["telegramLinked"] = true
		if e = t.Put(r.Context(), "user", u); e != nil {
			return e
		}
		return t.Delete(r.Context(), "telegram_link", str(link["id"]))
	})
	if e != nil {
		return e
	}
	respond(w, 200, safeUser(u))
	return nil
}
func (s *Server) botDigest(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "bot"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	var out Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		u, e := findTelegram(t, r.Context(), str(d["telegramId"]))
		if e != nil {
			return errStatus(404, "Start the bot first")
		}
		out, e = s.compose(t, r.Context(), u, "personal")
		if e != nil {
			return e
		}
		out["status"] = "sending"
		if e = t.Put(r.Context(), "digest", out); e != nil {
			return e
		}
		if e = s.enqueueDelivery(t, r.Context(), out, str(d["telegramId"]), str(u["id"])); e != nil {
			return e
		}
		key := hash(str(out["id"]) + ":" + str(d["telegramId"]))
		delivery, e := t.Get(r.Context(), "delivery", key)
		if e != nil {
			return e
		}
		delivery["status"] = "sending"
		delivery["manual"] = true
		delivery["attempts"] = 1
		delivery["leaseUntil"] = time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339)
		return t.Put(r.Context(), "delivery", delivery)
	})
	if e != nil {
		return e
	}
	respond(w, 200, out)
	return nil
}
func (s *Server) enqueueDelivery(t *Tx, ctx context.Context, d Doc, chatID, userID string) error {
	key := hash(str(d["id"]) + ":" + chatID)
	if _, e := t.Get(ctx, "delivery", key); e == nil {
		return nil
	}
	links := []any{}
	for _, raw := range array(d["articles"]) {
		a := object(raw)
		links = append(links, Doc{"label": localize(a["title"], locale(str(d["locale"]))), "url": s.articleLink(a, locale(str(d["locale"])))})
	}
	delivery := Doc{"id": key, "digestId": d["id"], "chatId": chatID, "userId": userID, "text": d["text"], "links": links, "status": "queued", "attempts": 0, "createdAt": now()}
	return t.Put(ctx, "delivery", delivery)
}
func (s *Server) claimDelivery(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "bot"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	if str(d["workerId"]) == "" {
		return errStatus(400, "workerId is required")
	}
	var found Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		all, e := t.List(r.Context(), "delivery")
		if e != nil {
			return e
		}
		for _, v := range all {
			if str(v["status"]) == "sending" && !parseTime(v["leaseUntil"]).After(time.Now()) {
				v["status"] = "unknown"
				v["error"] = "Delivery worker stopped; outcome requires manual review"
				if e = t.Put(r.Context(), "delivery", v); e != nil {
					return e
				}
				continue
			}
			if str(v["status"]) != "queued" || parseTime(v["retryAt"]).After(time.Now()) {
				continue
			}
			v["status"] = "sending"
			v["leaseToken"] = id() + id()
			v["leaseUntil"] = time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339)
			v["workerId"] = d["workerId"]
			v["attempts"] = integer(v["attempts"]) + 1
			found = v
			return t.Put(r.Context(), "delivery", v)
		}
		return nil
	})
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"delivery": found})
	return nil
}
func (s *Server) finishDelivery(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "bot"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		v, e := t.Get(r.Context(), "delivery", r.PathValue("id"))
		if e != nil {
			return e
		}
		if str(d["leaseToken"]) == "" || str(d["leaseToken"]) != str(v["leaseToken"]) {
			return errStatus(409, "Delivery lease is not owned")
		}
		if str(v["status"]) == "sent" && r.PathValue("action") == "complete" {
			return nil
		}
		if str(v["status"]) != "sending" {
			return errStatus(409, "Delivery is not sending")
		}
		switch r.PathValue("action") {
		case "complete":
			v["status"] = "sent"
			v["messageId"] = d["messageId"]
			v["sentAt"] = now()
			digest, e := t.Get(r.Context(), "digest", str(v["digestId"]))
			if e != nil {
				return e
			}
			digest["status"] = "sent"
			if e = t.Put(r.Context(), "digest", digest); e != nil {
				return e
			}
			for _, raw := range append(array(digest["articles"]), digest["outsideInterest"]) {
				a := object(raw)
				if str(a["id"]) == "" {
					continue
				}
				key := str(v["userId"]) + ":" + str(a["id"])
				if e = t.Put(r.Context(), "sent_item", Doc{"id": key, "userId": v["userId"], "articleId": a["id"], "sentAt": now()}); e != nil {
					return e
				}
			}
		case "fail":
			v["error"] = str(d["error"])
			if boolean(d["unknown"]) {
				v["status"] = "unknown"
			} else if integer(v["attempts"]) < 5 {
				v["status"] = "queued"
				seconds := integer(d["retryAfterSeconds"])
				if seconds < 30 {
					seconds = 30
				}
				v["retryAt"] = time.Now().Add(time.Duration(seconds) * time.Second).UTC().Format(time.RFC3339)
			} else {
				v["status"] = "failed"
			}
		default:
			return errStatus(400, "Unknown delivery action")
		}
		return t.Put(r.Context(), "delivery", v)
	})
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"ok": true})
	return nil
}
func (s *Server) scheduleDigests(t *Tx, ctx context.Context) error {
	settings, e := t.Get(ctx, "settings", "settings")
	if e != nil {
		return e
	}
	tz, e := time.LoadLocation(str(settings["timezone"]))
	if e != nil {
		return e
	}
	current := time.Now().In(tz)
	if current.Hour() < integer(settings["digestHour"]) {
		return nil
	}
	users, e := t.List(ctx, "user")
	if e != nil {
		return e
	}
	for _, u := range users {
		if str(u["telegramId"]) == "" || boolean(u["paused"]) {
			continue
		}
		if str(u["frequency"]) == "weekly" && current.Weekday() != time.Friday {
			continue
		}
		period := current.Format("2006-01-02")
		editionKey := hash(str(u["id"]) + ":" + period)
		if _, e = t.Get(ctx, "digest_period", editionKey); e == nil {
			continue
		}
		d, e := s.compose(t, ctx, u, "personal")
		if e != nil {
			return e
		}
		if len(array(d["articles"])) == 0 && len(array(d["opportunities"])) == 0 && len(object(d["outsideInterest"])) == 0 {
			continue
		}
		d["status"] = "queued"
		if e = t.Put(ctx, "digest", d); e != nil {
			return e
		}
		if e = s.enqueueDelivery(t, ctx, d, str(u["telegramId"]), str(u["id"])); e != nil {
			return e
		}
		if e = t.Put(ctx, "digest_period", Doc{"id": editionKey, "userId": u["id"], "digestId": d["id"], "period": period}); e != nil {
			return e
		}
	}
	return nil
}
