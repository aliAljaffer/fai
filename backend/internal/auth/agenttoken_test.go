package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAgentTokens(t *testing.T) {
	tokens, err := ParseAgentTokens("hermes:0123456789abcdef,claude:fedcba9876543210:work")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.Len() != 2 {
		t.Fatalf("len = %d", tokens.Len())
	}
	token, ok := tokens.Lookup("fedcba9876543210")
	if !ok || token.Name != "claude" || token.WorkspaceTag != "WORK" {
		t.Fatalf("lookup = %+v, %v", token, ok)
	}
	if _, ok := tokens.Lookup("wrong"); ok {
		t.Fatal("a wrong secret matched")
	}

	for _, raw := range []string{
		"no-secret-separator",
		"name:short",
		"a:0123456789abcdef,a:fedcba9876543210",
		":0123456789abcdef",
	} {
		if _, err := ParseAgentTokens(raw); err == nil {
			t.Errorf("%q was accepted", raw)
		}
	}
}

func TestAgentTokensMiddleware(t *testing.T) {
	tokens, err := ParseAgentTokens("hermes:0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	var subject string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject, _ = r.Context().Value(SubjectContextKey).(string)
		if agent, ok := AgentFromContext(r.Context()); ok {
			subject += "|" + agent.Name
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := tokens.Middleware(func(ctx context.Context, secret string) (AgentToken, bool) {
		if secret == "fai_from_database" {
			return AgentToken{Name: "from-db"}, true
		}
		return AgentToken{}, false
	}, nil)

	serve := func(header string) int {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		handler(next).ServeHTTP(rec, req)
		return rec.Code
	}

	if code := serve("Bearer 0123456789abcdef"); code != http.StatusOK || subject != "agent:hermes|hermes" {
		t.Fatalf("env key code=%d subject=%q", code, subject)
	}
	if code := serve("Bearer fai_from_database"); code != http.StatusOK || subject != "agent:from-db|from-db" {
		t.Fatalf("stored key code=%d subject=%q", code, subject)
	}
	if code := serve("Bearer nope"); code != http.StatusUnauthorized {
		t.Fatalf("bad key code = %d", code)
	}
	if code := serve(""); code != http.StatusUnauthorized {
		t.Fatalf("missing key code = %d", code)
	}
}

func TestAgentTokensMiddlewareFallsBackToSiteAuth(t *testing.T) {
	tokens, err := ParseAgentTokens("hermes:0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	fallback := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Fallback", "1")
			w.WriteHeader(http.StatusNoContent)
		})
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	tokens.Middleware(nil, fallback)(http.NotFoundHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Header().Get("X-Fallback") != "1" {
		t.Fatalf("fallback not used: code=%d", rec.Code)
	}
}
