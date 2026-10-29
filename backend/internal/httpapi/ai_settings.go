package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/ai"
	"github.com/alialjaffer/fai/backend/internal/store"
)

// aiProvider prefers saved database settings and falls back to the provider built
// from environment configuration. Reading on each call keeps changes live without
// a restart.
func (s *Server) aiProvider(ctx context.Context) ai.Provider {
	settings, err := s.Store.GetAISettings(ctx)
	if err == nil && strings.TrimSpace(settings.APIKey) != "" {
		if provider := ai.NewProvider(settings.Provider, settings.BaseURL, settings.APIKey, settings.Model); provider != nil {
			return provider
		}
	}
	return s.AI
}

type aiSettingsResponse struct {
	Provider   string     `json:"provider"`
	BaseURL    string     `json:"base_url"`
	Model      string     `json:"model"`
	HasAPIKey  bool       `json:"has_api_key"`
	Configured bool       `json:"configured"`
	Source     string     `json:"source"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

func (s *Server) aiSettingsResponse(ctx context.Context) (aiSettingsResponse, error) {
	settings := store.AISettings{
		Provider: ai.NormalizeProvider(s.AIProvider),
		BaseURL:  s.AIBaseURL,
		Model:    s.AIModel,
		APIKey:   s.AIAPIKey,
	}
	source := "environment"
	stored, err := s.Store.GetAISettings(ctx)
	switch {
	case err == nil:
		settings = stored
		source = "database"
	case errors.Is(err, store.ErrNotFound):
	default:
		return aiSettingsResponse{}, err
	}

	updatedAt := settings.UpdatedAt
	var updatedPtr *time.Time
	if source == "database" {
		updatedPtr = &updatedAt
	}
	return aiSettingsResponse{
		Provider:   ai.NormalizeProvider(settings.Provider),
		BaseURL:    settings.BaseURL,
		Model:      settings.Model,
		HasAPIKey:  strings.TrimSpace(settings.APIKey) != "",
		Configured: s.aiProvider(ctx) != nil,
		Source:     source,
		UpdatedAt:  updatedPtr,
	}, nil
}

func (s *Server) getAISettings(w http.ResponseWriter, r *http.Request) {
	response, err := s.aiSettingsResponse(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

type aiSettingsBody struct {
	Provider    string `json:"provider"`
	BaseURL     string `json:"base_url"`
	Model       string `json:"model"`
	APIKey      string `json:"api_key"`
	ClearAPIKey bool   `json:"clear_api_key"`
}

func validAIProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", ai.ProviderAnthropic, ai.ProviderOpenAI, ai.ProviderOpenAICompat, "opencode-zen":
		return true
	default:
		return false
	}
}

func validAIBaseURL(raw string) bool {
	if raw == "" {
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func (s *Server) updateAISettings(w http.ResponseWriter, r *http.Request) {
	var body aiSettingsBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if !validAIProvider(body.Provider) {
		writeError(w, http.StatusBadRequest, "provider must be anthropic, openai, or openai-compat")
		return
	}

	baseURL := strings.TrimRight(strings.TrimSpace(body.BaseURL), "/")
	if !validAIBaseURL(baseURL) {
		writeError(w, http.StatusBadRequest, "base_url must be an http or https URL")
		return
	}

	ctx := r.Context()
	apiKey := strings.TrimSpace(body.APIKey)
	if apiKey == "" && !body.ClearAPIKey {
		if stored, err := s.Store.GetAISettings(ctx); err == nil {
			apiKey = stored.APIKey
		} else {
			apiKey = s.AIAPIKey
		}
	}
	if apiKey == "" && !body.ClearAPIKey {
		writeError(w, http.StatusBadRequest, "api_key is required")
		return
	}

	saved, err := s.Store.SaveAISettings(ctx, store.AISettings{
		Provider: ai.NormalizeProvider(body.Provider),
		BaseURL:  baseURL,
		Model:    strings.TrimSpace(body.Model),
		APIKey:   apiKey,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	updatedAt := saved.UpdatedAt
	writeJSON(w, http.StatusOK, aiSettingsResponse{
		Provider:   saved.Provider,
		BaseURL:    saved.BaseURL,
		Model:      saved.Model,
		HasAPIKey:  strings.TrimSpace(saved.APIKey) != "",
		Configured: s.aiProvider(ctx) != nil,
		Source:     "database",
		UpdatedAt:  &updatedAt,
	})
}
