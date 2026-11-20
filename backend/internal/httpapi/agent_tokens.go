package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/auth"
	"github.com/alialjaffer/fai/backend/internal/store"
)

const agentSecretPrefix = "fai_"

func hashAgentSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func newAgentSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return agentSecretPrefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

// displayAgentPrefix is what the settings page shows so a key can be told apart
// without revealing it.
func displayAgentPrefix(secret string) string {
	return secret[:len(agentSecretPrefix)+8] + "…"
}

// resolveAgentToken looks up a key created in the UI. Environment keys are
// checked by auth.AgentTokens before this runs.
func (s *Server) resolveAgentToken(ctx context.Context, secret string) (auth.AgentToken, bool) {
	if !strings.HasPrefix(secret, agentSecretPrefix) {
		return auth.AgentToken{}, false
	}
	row, err := s.Store.AgentTokenByHash(ctx, hashAgentSecret(secret))
	if err != nil {
		return auth.AgentToken{}, false
	}
	token := auth.AgentToken{Name: row.Name}
	if row.WorkspaceID != nil {
		if workspace, err := s.Store.GetWorkspace(ctx, *row.WorkspaceID); err == nil {
			token.WorkspaceTag = workspace.Tag
		}
	}
	s.touchAgentToken(row.ID)
	return token, true
}

// touchAgentToken records use at most once a minute per key, so a chatty agent
// turn does not turn every tool call into a database write.
func (s *Server) touchAgentToken(id int64) {
	s.agentTouchMu.Lock()
	if last, ok := s.agentTouched[id]; ok && time.Since(last) < time.Minute {
		s.agentTouchMu.Unlock()
		return
	}
	if s.agentTouched == nil {
		s.agentTouched = make(map[int64]time.Time)
	}
	s.agentTouched[id] = time.Now()
	s.agentTouchMu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Store.TouchAgentToken(ctx, id, time.Now())
	}()
}

func (s *Server) listAgentTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := s.Store.ListAgentTokens(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

type agentTokenBody struct {
	Name        string `json:"name"`
	WorkspaceID *int64 `json:"workspace_id"`
}

type createdAgentToken struct {
	Token  store.AgentToken `json:"token"`
	Secret string           `json:"secret"`
}

func (s *Server) createAgentToken(w http.ResponseWriter, r *http.Request) {
	var body agentTokenBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || len(name) > 64 {
		writeError(w, http.StatusBadRequest, "name is required and must be 64 characters or fewer")
		return
	}
	if body.WorkspaceID != nil {
		if _, err := s.Store.GetWorkspace(r.Context(), *body.WorkspaceID); err != nil {
			writeError(w, http.StatusBadRequest, "workspace not found")
			return
		}
	}
	secret, err := newAgentSecret()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate a key")
		return
	}
	token, err := s.Store.CreateAgentToken(r.Context(), name, hashAgentSecret(secret), displayAgentPrefix(secret), body.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, createdAgentToken{Token: token, Secret: secret})
}

func (s *Server) deleteAgentToken(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "tokenID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid token id")
		return
	}
	if err := s.Store.DeleteAgentToken(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "agent key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.agentTouchMu.Lock()
	delete(s.agentTouched, id)
	s.agentTouchMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}
