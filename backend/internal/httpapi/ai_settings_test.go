package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store/lite"
)

func sendJSON(t *testing.T, method, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestAISettingsPersistAndOverrideEnvironment(t *testing.T) {
	st, err := lite.New(filepath.Join(t.TempDir(), "fai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	server := httptest.NewServer(NewRouter(&Server{
		Store:      st,
		AI:         &fakeAI{},
		AIProvider: "anthropic",
		AIModel:    "env-model",
		AIAPIKey:   "env-key",
	}))
	defer server.Close()

	res := sendJSON(t, http.MethodGet, server.URL+"/api/ai/settings", "")
	var settings aiSettingsResponse
	if err := json.NewDecoder(res.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if settings.Source != "environment" || !settings.HasAPIKey || settings.Model != "env-model" {
		t.Fatalf("environment settings = %#v", settings)
	}

	res = sendJSON(t, http.MethodPut, server.URL+"/api/ai/settings",
		`{"provider":"openai","base_url":"https://example.test/v1","model":"gpt-test","api_key":"sk-test"}`)
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", res.StatusCode, raw)
	}
	if strings.Contains(string(raw), "sk-test") {
		t.Fatalf("response leaked the API key: %s", raw)
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Source != "database" || settings.Provider != "openai" || settings.Model != "gpt-test" {
		t.Fatalf("saved settings = %#v", settings)
	}

	stored, err := st.GetAISettings(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if stored.Provider != "openai" || stored.BaseURL != "https://example.test/v1" || stored.APIKey != "sk-test" {
		t.Fatalf("stored settings = %#v", stored)
	}

	res = sendJSON(t, http.MethodPut, server.URL+"/api/ai/settings",
		`{"provider":"anthropic","base_url":"","model":"claude-test"}`)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("blank key update status = %d", res.StatusCode)
	}
	stored, err = st.GetAISettings(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if stored.APIKey != "sk-test" || stored.Provider != "anthropic" || stored.Model != "claude-test" {
		t.Fatalf("retained settings = %#v", stored)
	}

	res = sendJSON(t, http.MethodPut, server.URL+"/api/ai/settings",
		`{"provider":"unknown","base_url":"","model":""}`)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown provider status = %d, want 400", res.StatusCode)
	}

	res = sendJSON(t, http.MethodPut, server.URL+"/api/ai/settings",
		`{"provider":"openai","base_url":"file:///etc/passwd","model":""}`)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid base URL status = %d, want 400", res.StatusCode)
	}

	res = sendJSON(t, http.MethodGet, server.URL+"/api/ai/status", "")
	var status struct {
		Configured bool `json:"configured"`
	}
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if !status.Configured {
		t.Error("saved provider is not reported as configured")
	}
}
