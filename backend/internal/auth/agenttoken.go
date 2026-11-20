package auth

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
)

// AgentToken identifies a non-browser client (an AI agent) that authenticates
// with a static bearer token from FAI_AGENT_TOKENS.
type AgentToken struct {
	Name string
	// WorkspaceTag scopes the token to one workspace. Empty grants every workspace.
	WorkspaceTag string
}

type agentTokenSecret struct {
	token  AgentToken
	secret string
}

// AgentTokens is the configured set of agent bearer tokens.
type AgentTokens struct {
	entries []agentTokenSecret
}

// ParseAgentTokens reads comma-separated "name:secret[:workspace_tag]" entries.
// Secrets and workspace tags may not contain a colon.
func ParseAgentTokens(raw string) (AgentTokens, error) {
	var tokens AgentTokens
	seen := make(map[string]bool)
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return AgentTokens{}, fmt.Errorf("FAI_AGENT_TOKENS entry %q must be name:secret[:workspace_tag]", parts[0])
		}
		name := strings.TrimSpace(parts[0])
		secret := strings.TrimSpace(parts[1])
		tag := ""
		if len(parts) == 3 {
			tag = strings.ToUpper(strings.TrimSpace(parts[2]))
		}
		if name == "" || strings.ContainsAny(name, " \t") {
			return AgentTokens{}, fmt.Errorf("FAI_AGENT_TOKENS entry %q has an invalid name", parts[0])
		}
		if len(secret) < 16 {
			return AgentTokens{}, fmt.Errorf("FAI_AGENT_TOKENS entry %q needs a secret of at least 16 characters", name)
		}
		if seen[name] {
			return AgentTokens{}, fmt.Errorf("FAI_AGENT_TOKENS has two entries named %q", name)
		}
		seen[name] = true
		tokens.entries = append(tokens.entries, agentTokenSecret{
			token:  AgentToken{Name: name, WorkspaceTag: tag},
			secret: secret,
		})
	}
	return tokens, nil
}

// Len reports how many agent tokens are configured.
func (t AgentTokens) Len() int {
	return len(t.entries)
}

// Lookup returns the agent for a presented secret. Every entry is compared in
// constant time so timing does not reveal which secret matched.
func (t AgentTokens) Lookup(secret string) (AgentToken, bool) {
	var found AgentToken
	matched := 0
	for _, entry := range t.entries {
		if subtle.ConstantTimeCompare([]byte(secret), []byte(entry.secret)) == 1 {
			found, matched = entry.token, 1
		}
	}
	return found, matched == 1
}

// AgentContextKey is set by Middleware on an authenticated agent request.
const AgentContextKey contextKey = "agent_token"

// AgentFromContext returns the agent that authenticated the request.
func AgentFromContext(ctx context.Context) (AgentToken, bool) {
	token, ok := ctx.Value(AgentContextKey).(AgentToken)
	return token, ok
}

// AgentTokenResolver looks up a secret that is not in the environment set, such
// as a key created in the UI and stored in the database.
type AgentTokenResolver func(ctx context.Context, secret string) (AgentToken, bool)

// Middleware authenticates agent bearer tokens and otherwise defers to the
// configured auth middleware. It is used for /mcp, which the reverse proxy
// exempts from Basic auth because an agent cannot complete an interactive login.
func (t AgentTokens) Middleware(resolve AgentTokenResolver, fallback func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
				secret := strings.TrimSpace(raw)
				if token, found := t.Lookup(secret); found {
					next.ServeHTTP(w, r.WithContext(agentContext(r.Context(), token)))
					return
				}
				if resolve != nil {
					if token, found := resolve(r.Context(), secret); found {
						next.ServeHTTP(w, r.WithContext(agentContext(r.Context(), token)))
						return
					}
				}
			}
			if fallback != nil {
				fallback(next).ServeHTTP(w, r)
				return
			}
			http.Error(w, "invalid agent token", http.StatusUnauthorized)
		})
	}
}

func agentContext(ctx context.Context, token AgentToken) context.Context {
	ctx = context.WithValue(ctx, SubjectContextKey, "agent:"+token.Name)
	return context.WithValue(ctx, AgentContextKey, token)
}
