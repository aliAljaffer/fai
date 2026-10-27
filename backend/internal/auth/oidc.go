package auth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type contextKey string

// SubjectContextKey is set by Middleware on a verified request's context so
// downstream handlers (e.g. the AI rate limiter) can key per-user.
const SubjectContextKey contextKey = "jwt_subject"

// claims mirrors the subset of a Keycloak access token this app needs. Keycloak does
// not put the requesting client into `aud` unless a per-client audience mapper is
// configured; `azp` (authorized party) is the field it reliably sets instead. Rather
// than depend on an audience mapper existing, Middleware accepts either aud or azp
// matching the configured audience.
type claims struct {
	Subject         string `json:"sub"`
	RawAudience     any    `json:"aud"`
	AuthorizedParty string `json:"azp"`
}

func (c *claims) audiences() []string {
	switch v := c.RawAudience.(type) {
	case string:
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, a := range v {
			if s, ok := a.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// Middleware validates a bearer JWT's signature, exp, and iss via JWKS discovery
// against issuerURL, then requires audience (matching either `aud` or `azp`).
// Returns an error if the issuer's discovery document can't be fetched, so the
// caller can fail startup loudly rather than silently accepting unverifiable tokens.
func Middleware(ctx context.Context, issuerURL, audience string) (func(http.Handler) http.Handler, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{SkipClientIDCheck: true})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			rawToken, ok := strings.CutPrefix(authHeader, "Bearer ")
			if !ok || rawToken == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			idToken, err := verifier.Verify(r.Context(), rawToken)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			var c claims
			if err := idToken.Claims(&c); err != nil {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}

			audOK := c.AuthorizedParty == audience
			for _, a := range c.audiences() {
				if a == audience {
					audOK = true
				}
			}
			if !audOK {
				http.Error(w, "invalid audience", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), SubjectContextKey, c.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}

// MustMiddleware is a convenience wrapper for main.go: logs and returns a
// fail-closed (always-401) middleware if OIDC discovery fails at startup,
// rather than crashing the whole process over a transient network issue.
func MustMiddleware(ctx context.Context, issuerURL, audience string) func(http.Handler) http.Handler {
	mw, err := Middleware(ctx, issuerURL, audience)
	if err != nil {
		log.Printf("auth: failed to initialize OIDC provider at %s: %v — all /api requests will 401 until this is fixed", issuerURL, err)
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeAuthError(w)
			})
		}
	}
	return mw
}

func writeAuthError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "auth not available"})
}
