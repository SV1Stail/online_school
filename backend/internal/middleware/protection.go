package middleware

import (
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	last     time.Time
	lastSeen time.Time
}

type RateLimiter struct {
	mu         sync.Mutex
	clients    map[string]*bucket
	rate       float64
	burst      float64
	maxClients int
	lastClean  time.Time
}

func NewRateLimiter(rate float64, burst, maxClients int) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		clients:    make(map[string]*bucket),
		rate:       rate,
		burst:      float64(burst),
		maxClients: maxClients,
		lastClean:  now,
	}
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !l.allow(ClientIPFromContext(request.Context()), time.Now()) {
			writer.Header().Set("Retry-After", "1")
			http.Error(writer, `{"error":{"code":"rate_limited","message":"too many requests"}}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (l *RateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastClean) >= time.Minute {
		l.cleanup(now)
	}
	current, exists := l.clients[key]
	if !exists {
		if len(l.clients) >= l.maxClients {
			return false
		}
		current = &bucket{tokens: l.burst, last: now, lastSeen: now}
		l.clients[key] = current
	}
	current.tokens += now.Sub(current.last).Seconds() * l.rate
	if current.tokens > l.burst {
		current.tokens = l.burst
	}
	current.last = now
	current.lastSeen = now
	if current.tokens < 1 {
		return false
	}
	current.tokens--
	return true
}

func (l *RateLimiter) cleanup(now time.Time) {
	for key, current := range l.clients {
		if now.Sub(current.lastSeen) > 10*time.Minute {
			delete(l.clients, key)
		}
	}
	l.lastClean = now
}

func ConcurrencyLimit(limit int) func(http.Handler) http.Handler {
	semaphore := make(chan struct{}, limit)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
				next.ServeHTTP(writer, request)
			default:
				writer.Header().Set("Retry-After", "1")
				http.Error(writer, `{"error":{"code":"server_busy","message":"server is busy"}}`, http.StatusServiceUnavailable)
			}
		})
	}
}

func Timeout(duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(
			next,
			duration,
			`{"error":{"code":"request_timeout","message":"request timed out"}}`,
		)
	}
}
