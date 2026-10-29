package telegram

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
