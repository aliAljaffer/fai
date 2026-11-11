package telegram

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestTranscodeToMP3(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	source := filepath.Join(t.TempDir(), "voice.ogg")
	generate := exec.Command("ffmpeg",
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=0.3",
		"-c:a", "libopus", "-f", "ogg", source,
	)
	if err := generate.Run(); err != nil {
		t.Skipf("cannot generate OGG fixture: %v", err)
	}
	audio, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}

	mp3, err := transcodeToMP3(context.Background(), audio)
	if err != nil {
		t.Fatalf("transcode: %v", err)
	}
	if len(mp3) == 0 {
		t.Fatal("transcode returned no audio")
	}
	if !bytes.HasPrefix(mp3, []byte("ID3")) && !(mp3[0] == 0xFF && mp3[1]&0xE0 == 0xE0) {
		t.Fatalf("output is not MP3: % x", mp3[:min(4, len(mp3))])
	}
}

type chunkRunner struct {
	ChatRunner
	mu    sync.Mutex
	sizes []int
}

func (r *chunkRunner) Transcribe(_ context.Context, audio []byte, format string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sizes = append(r.sizes, len(audio))
	if format != "mp3" {
		return "", fmt.Errorf("format %q", format)
	}
	return fmt.Sprintf(" part%d ", len(r.sizes)), nil
}

func TestTranscribeChunksSplitsLongAudio(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	source := filepath.Join(t.TempDir(), "voice.ogg")
	generate := exec.Command("ffmpeg",
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=150",
		"-c:a", "libopus", "-f", "ogg", source,
	)
	if err := generate.Run(); err != nil {
		t.Skipf("cannot generate OGG fixture: %v", err)
	}
	audio, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}

	runner := &chunkRunner{}
	got, err := (&Bot{runner: runner}).transcribeChunks(context.Background(), audio, 150)
	if err != nil {
		t.Fatalf("transcribe: %v", err)
	}
	if len(runner.sizes) != 3 {
		t.Fatalf("got %d chunks, want 3", len(runner.sizes))
	}
	if strings.Count(got, "part") != 3 || strings.Contains(got, "  ") {
		t.Fatalf("joined transcript %q", got)
	}
	if last := runner.sizes; slices.Min(last) == slices.Max(last) {
		t.Fatalf("chunk sizes %v, want a shorter final chunk", last)
	}
}
