package transcribe

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTranscribeSendsMultipartRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/speech-to-text" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("xi-api-key"); got != "secret" {
			t.Errorf("xi-api-key = %q", got)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if got := r.FormValue("model_id"); got != "scribe_v2" {
			t.Errorf("model_id = %q", got)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if header.Filename != "voice.m4a" {
			t.Errorf("filename = %q", header.Filename)
		}
		if got := header.Header.Get("Content-Type"); got != "audio/mp4" {
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

	client := newElevenLabs("secret", "", server.URL)
	got, err := client.Transcribe(context.Background(), []byte("audio-bytes"), "m4a")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Fatalf("transcript = %q", got)
	}
}

func TestTranscribeReportsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":{"message":"missing speech_to_text permission"}}`))
	}))
	defer server.Close()

	client := newElevenLabs("secret", "", server.URL)
	_, err := client.Transcribe(context.Background(), []byte("audio"), "ogg")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "missing speech_to_text permission") {
		t.Fatalf("error = %v", err)
	}
}

func TestTranscribeRequiresKeyAndAudio(t *testing.T) {
	if _, err := newElevenLabs("", "", "http://example.test").Transcribe(context.Background(), []byte("audio"), "ogg"); err == nil {
		t.Fatal("expected a missing key error")
	}
	if _, err := newElevenLabs("secret", "", "http://example.test").Transcribe(context.Background(), nil, "ogg"); err == nil {
		t.Fatal("expected a missing audio error")
	}
}
