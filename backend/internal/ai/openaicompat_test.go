package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenCodeGoProviderUsesInferenceBaseURL(t *testing.T) {
	provider := NewProvider(ProviderOpenCodeGo, "", "key", "")
	if provider == nil {
		t.Fatal("provider is nil")
	}
	impl, ok := provider.(*openAICompatProvider)
	if !ok {
		t.Fatalf("provider type = %T", provider)
	}
	if impl.baseURL != openCodeGoBaseURL {
		t.Fatalf("base URL = %q", impl.baseURL)
	}
}

func TestTranscribeSendsAudioContentParts(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Errorf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":" Buy milk "}}]}`))
	}))
	defer server.Close()

	provider := newOpenAICompat(server.URL, "", "key", "")
	text, err := provider.Transcribe(context.Background(), []byte("audio"), "ogg")
	if err != nil {
		t.Fatal(err)
	}
	if text != "Buy milk" {
		t.Fatalf("text = %q", text)
	}
	if request["model"] != transcriptionModel {
		t.Errorf("model = %v", request["model"])
	}
	if request["temperature"] != float64(0) {
		t.Errorf("temperature = %v", request["temperature"])
	}

	messages, _ := request["messages"].([]any)
	if len(messages) != 2 {
		t.Fatalf("messages = %#v", messages)
	}
	user, _ := messages[1].(map[string]any)
	parts, _ := user["content"].([]any)
	if len(parts) != 2 {
		t.Fatalf("parts = %#v", parts)
	}
	audioPart, _ := parts[0].(map[string]any)
	if audioPart["type"] != "input_audio" {
		t.Fatalf("audio part = %#v", audioPart)
	}
	input, _ := audioPart["input_audio"].(map[string]any)
	if input["data"] != "YXVkaW8=" || input["format"] != "ogg" {
		t.Errorf("input audio = %#v", input)
	}
	textPart, _ := parts[1].(map[string]any)
	if textPart["type"] != "text" || !strings.Contains(textPart["text"].(string), "Transcribe") {
		t.Errorf("text part = %#v", textPart)
	}
}
