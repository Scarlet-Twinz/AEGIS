package aegis

import (
	"sync"
	"time"
)

type clientBucket struct { tokens float64; updated time.Time }
type Limiter struct { mu sync.Mutex; clients map[string]clientBucket; rate, burst float64 }

func NewLimiter(requestsPerMinute, burst int) *Limiter { return &Limiter{clients: make(map[string]clientBucket), rate: float64(requestsPerMinute)/60, burst: float64(burst)} }
func (l *Limiter) Allow(key string, now time.Time) bool {
	l.mu.Lock(); defer l.mu.Unlock()
	b := l.clients[key]
	if b.updated.IsZero() { b = clientBucket{tokens:l.burst, updated:now} }
	b.tokens = min(l.burst, b.tokens+now.Sub(b.updated).Seconds()*l.rate); b.updated = now
	if b.tokens < 1 { l.clients[key] = b; return false }
	b.tokens--; l.clients[key] = b; return true
}
func min(a,b float64) float64 { if a < b { return a }; return b }
