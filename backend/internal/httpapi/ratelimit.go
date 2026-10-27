package httpapi

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// rateLimitByUser enforces a per-user token bucket on expensive AI endpoints, since fai
// is publicly exposed (see plan §3) and a valid bearer token could otherwise burn
// through the configured AI API key with repeated requests. Keyed by the request's JWT
// subject once auth (phase 13) sets one on the context; falls back to a single shared
// bucket under DISABLE_AUTH, which is fine for local/personal use.
func rateLimitByUser(requestsPerMinute float64, burst int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	limiters := map[string]*rate.Limiter{}

	limiterFor := func(key string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		l, ok := limiters[key]
		if !ok {
			l = rate.NewLimiter(rate.Limit(requestsPerMinute/60), burst)
			limiters[key] = l
		}
		return l
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := "anonymous"
			if sub, ok := r.Context().Value(jwtSubjectKey).(string); ok && sub != "" {
				key = sub
			}
			if !limiterFor(key).Allow() {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type contextKey string

const jwtSubjectKey contextKey = "jwt_subject"
