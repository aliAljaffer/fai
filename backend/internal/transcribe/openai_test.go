package transcribe

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAICompatSendsMultipartRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/audio/transcriptions" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q", got)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if got := r.FormValue("model"); got != "small.en" {
			t.Errorf("model = %q", got)
		}
		if got := r.FormValue("language"); got != "en" {
			t.Errorf("language = %q", got)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if header.Filename != "voice.mp3" {
			t.Errorf("filename = %q", header.Filename)
		}
		if got := header.Header.Get("Content-Type"); got != "audio/mpeg" {
			t.Errorf("content type = %q", got)
		}
		audio, err := io.ReadAll(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(audio) != "audio-bytes" {
			t.Errorf("audio = %q", audio)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"  hello world  "}`))
	}))
	defer server.Close()

	client := NewOpenAICompat(server.URL+"/v1", "small.en", "secret", "en")
	got, err := client.Transcribe(context.Background(), []byte("audio-bytes"), "mp3")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Fatalf("transcript = %q", got)
	}
}

func TestOpenAICompatReportsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model not found"}}`))
	}))
	defer server.Close()

	client := NewOpenAICompat(server.URL, "missing", "", "")
	_, err := client.Transcribe(context.Background(), []byte("audio"), "ogg")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("error = %v", err)
	}
}
