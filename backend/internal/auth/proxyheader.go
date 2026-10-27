package auth

import (
	"context"
	"net/http"
)

// userHeaders covers both ways oauth2-proxy is commonly wired in front of a backend:
// X-Forwarded-User/-Email when it runs as the reverse proxy itself (--upstream mode),
// X-Auth-Request-User/-Email when a separate proxy (e.g. Caddy's forward_auth) calls
// its /oauth2/auth endpoint and copies the response headers upstream.
var userHeaders = []string{"X-Forwarded-User", "X-Forwarded-Email", "X-Auth-Request-User", "X-Auth-Request-Email"}

// ProxyHeaderMiddleware trusts a user identity header set by a fronting oauth2-proxy,
// used by the docker-compose deployment instead of validating a JWT itself. This is
// only safe when the backend is unreachable except through that proxy —
// docker-compose enforces this via its network, keeping the backend's port
// unpublished to the host and reachable only from other containers. A request with
// none of the known headers is rejected, since their absence on a supposedly
// proxy-fronted request means the proxy isn't actually in front of this request.
func ProxyHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user string
		for _, h := range userHeaders {
			if v := r.Header.Get(h); v != "" {
				user = v
				break
			}
		}
		if user == "" {
			http.Error(w, "missing authenticated user header", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), SubjectContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
