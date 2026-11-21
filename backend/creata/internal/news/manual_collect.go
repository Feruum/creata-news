package news

import (
	"net"
	"net/http"
	"net/url"
)

func (s *Server) manualCollect(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !has(u["roles"], "author") && !has(u["roles"], "editor") && !has(u["roles"], "chief_editor") {
		return errStatus(403, "Editorial submission denied")
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	link, e := canonicalURL(str(d["url"]))
	if e != nil {
		return errStatus(400, "Enter a valid public HTTP(S) article URL")
	}
	parsed, _ := url.Parse(link)
	if !s.Config.AllowPrivateSources {
		ip := net.ParseIP(parsed.Hostname())
		if isLoopback(parsed.Hostname()) || (ip != nil && (!ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast())) {
			return errStatus(400, "Private source targets are forbidden")
		}
	}
	lang := str(d["locale"])
	if lang == "" {
		lang = str(u["locale"])
	}
	if !has([]string{"ru", "kk", "en"}, lang) {
		return errStatus(400, "Unsupported source language")
	}
	if len(array(d["topics"])) > 12 {
		return errStatus(400, "Too many source topics")
	}
	source := Doc{"id": id(), "name": "Editorial link Â· " + parsed.Hostname(), "type": "website", "url": link, "locale": lang, "topics": array(d["topics"]), "intervalMinutes": 1440, "active": false, "singlePage": true, "oneoff": true, "createdBy": u["id"], "createdAt": now()}
	job := newJob("collect", Doc{"source": source, "actorId": u["id"]})
	job["createdBy"] = u["id"]
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		if e := t.Put(r.Context(), "source", source); e != nil {
			return e
		}
		if e := t.Put(r.Context(), "job", job); e != nil {
			return e
		}
		return t.Audit(r.Context(), u, "bank.collect", str(job["id"]), Doc{"url": link, "sourceId": source["id"]})
	})
	if e != nil {
		return e
	}
	respond(w, 202, Doc{"job": job})
	return nil
}
