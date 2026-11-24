package news

import (
	"crypto/hmac"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
)

func parseTrustedProxies(value string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(item)
		if err != nil || prefix.Bits() == 0 || prefix.Addr().Is4In6() {
			return nil, fmt.Errorf("CREATA_TRUSTED_PROXIES requires explicit IPv4 or IPv6 CIDRs")
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

func (s *Server) trustsProxy(addr netip.Addr) bool {
	for _, prefix := range s.trustedProxies {
		if prefix.Contains(addr.Unmap().WithZone("")) {
			return true
		}
	}
	return false
}

func (s *Server) requestClient(r *http.Request) string {
	remote := r.RemoteAddr
	if peer, err := netip.ParseAddrPort(remote); err == nil {
		remote = peer.Addr().String()
	}
	addr, err := netip.ParseAddr(remote)
	if err != nil {
		return remote
	}
	peer := addr.Unmap().WithZone("").String()
	if !s.trustsProxy(addr) {
		return peer
	}
	chain := strings.Join(r.Header.Values("X-Forwarded-For"), ",")
	if len(chain) > 4096 {
		return peer
	}
	for chain != "" {
		hop := chain
		if last := strings.LastIndexByte(chain, ','); last >= 0 {
			chain, hop = chain[:last], chain[last+1:]
		} else {
			chain = ""
		}
		forwarded, err := netip.ParseAddr(strings.TrimSpace(hop))
		if err != nil || forwarded.Zone() != "" {
			return peer
		}
		if !s.trustsProxy(forwarded) {
			return forwarded.Unmap().String()
		}
	}
	return peer
}

func (s *Server) allowRequest(r *http.Request, now time.Time) bool {
	read := r.Method == http.MethodGet || r.Method == http.MethodHead
	path := r.URL.Path
	if read && strings.HasPrefix(path, base+"/public/") && s.Config.SSRToken != "" &&
		hmac.Equal([]byte(s.Config.SSRToken), []byte(r.Header.Get("X-Creata-SSR-Token"))) {
		return s.ssrLimiter.allow("public-ssr", now)
	}
	client := s.requestClient(r)
	if strings.HasPrefix(path, base+"/auth/") && path != base+"/auth/me" {
		return s.authLimiter.allow(client, now)
	}
	if read {
		return s.readLimiter.allow(client, now)
	}
	return s.writeLimiter.allow(client, now)
}

type burstEntry struct {
	window time.Time
	used   int
}
type burstLimiter struct {
	mu       sync.Mutex
	limit    int
	duration time.Duration
	entries  map[string]burstEntry
}

func newBurstLimiter(limit int, duration time.Duration) *burstLimiter {
	return &burstLimiter{limit: limit, duration: duration, entries: map[string]burstEntry{}}
}
func (l *burstLimiter) allow(key string, at time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.entries) > 10000 {
		for k, v := range l.entries {
			if !at.Before(v.window.Add(l.duration)) {
				delete(l.entries, k)
			}
		}
		if len(l.entries) > 10000 {
			return false
		}
	}
	v := l.entries[key]
	if v.window.IsZero() || !at.Before(v.window.Add(l.duration)) {
		v = burstEntry{window: at}
	}
	if v.used >= l.limit {
		return false
	}
	v.used++
	l.entries[key] = v
	return true
}
