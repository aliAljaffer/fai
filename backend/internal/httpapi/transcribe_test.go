package httpapi

import (
	"context"
	"testing"
)

type fakeTranscriber struct {
	text   string
	format string
}

func (f *fakeTranscriber) Transcribe(_ context.Context, _ []byte, format string) (string, error) {
	f.format = format
	return f.text, nil
}

func TestTranscribeUsesConfiguredTranscriber(t *testing.T) {
	transcriber := &fakeTranscriber{text: "hello"}
	server := &Server{Transcriber: transcriber}
	got, err := server.Transcribe(context.Background(), []byte("audio"), "ogg")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Fatalf("transcript = %q", got)
	}
	if transcriber.format != "ogg" {
		t.Fatalf("format = %q", transcriber.format)
	}
}
