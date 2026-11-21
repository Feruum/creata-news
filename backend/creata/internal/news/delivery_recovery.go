package news

import (
	"context"
	"net/http"
)

func (s *Server) markSent(t *Tx, ctx context.Context, v Doc, messageID any) error {
	v["status"] = "sent"
	v["messageId"] = messageID
	v["sentAt"] = now()
	digest, e := t.Get(ctx, "digest", str(v["digestId"]))
	if e != nil {
		return e
	}
	digest["status"] = "sent"
	if e = t.Put(ctx, "digest", digest); e != nil {
		return e
	}
	for _, raw := range append(array(digest["articles"]), digest["outsideInterest"]) {
		a := object(raw)
		if str(a["id"]) == "" {
			continue
		}
		key := str(v["userId"]) + ":" + str(a["id"])
		if e = t.Put(ctx, "sent_item", Doc{"id": key, "userId": v["userId"], "articleId": a["id"], "sentAt": now()}); e != nil {
			return e
		}
	}
	return t.Put(ctx, "delivery", v)
}
func (s *Server) manualDeliveryResult(w http.ResponseWriter, r *http.Request) error {
	if e := s.service(r, "bot"); e != nil {
		return e
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		key := hash(r.PathValue("id") + ":" + str(d["telegramId"]))
		v, e := t.Get(r.Context(), "delivery", key)
		if e != nil {
			return e
		}
		if !boolean(v["manual"]) {
			return errStatus(400, "Not a manual delivery")
		}
		if str(v["status"]) == "sent" && r.PathValue("action") == "complete" {
			return nil
		}
		if !has([]string{"sending", "unknown"}, str(v["status"])) {
			return errStatus(409, "Manual delivery is already resolved")
		}
		switch r.PathValue("action") {
		case "complete":
			return s.markSent(t, r.Context(), v, d["messageId"])
		case "fail":
			v["status"] = "failed"
			if boolean(d["unknown"]) {
				v["status"] = "unknown"
			}
			v["error"] = str(d["error"])
			digest, e := t.Get(r.Context(), "digest", str(v["digestId"]))
			if e != nil {
				return e
			}
			digest["status"] = v["status"]
			if e = t.Put(r.Context(), "digest", digest); e != nil {
				return e
			}
			return t.Put(r.Context(), "delivery", v)
		default:
			return errStatus(400, "Unknown manual delivery result")
		}
	})
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"ok": true})
	return nil
}
func (s *Server) deliveryList(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !has(u["roles"], "admin") && !has(u["roles"], "chief_editor") && !has(u["roles"], "editor") {
		return errStatus(403, "Delivery access denied")
	}
	items, e := s.Store.List(r.Context(), "delivery")
	if e != nil {
		return e
	}
	for _, v := range items {
		delete(v, "leaseToken")
	}
	respond(w, 200, Doc{"items": items})
	return nil
}
func (s *Server) deliveryResolve(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !has(u["roles"], "admin") && !has(u["roles"], "chief_editor") {
		return errStatus(403, "Delivery recovery denied")
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	var v Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		v, e = t.Get(r.Context(), "delivery", r.PathValue("id"))
		if e != nil {
			return e
		}
		if !has([]string{"unknown", "failed"}, str(v["status"])) {
			return errStatus(409, "Only failed or unknown deliveries can be resolved")
		}
		switch str(d["action"]) {
		case "retry":
			v["status"] = "queued"
			v["manual"] = false
			v["attempts"] = 0
			delete(v, "retryAt")
			delete(v, "leaseToken")
			if e = t.Put(r.Context(), "delivery", v); e != nil {
				return e
			}
		case "mark_sent":
			if e = s.markSent(t, r.Context(), v, d["messageId"]); e != nil {
				return e
			}
		default:
			return errStatus(400, "Choose retry or mark_sent")
		}
		return t.Audit(r.Context(), u, "delivery.resolve", str(v["id"]), Doc{"action": d["action"]})
	})
	if e != nil {
		return e
	}
	respond(w, 200, v)
	return nil
}
