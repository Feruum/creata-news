package news

import (
	"sync"
	"time"
)

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
