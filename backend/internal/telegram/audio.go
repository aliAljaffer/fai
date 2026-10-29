package telegram

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const transcodeTimeout = 30 * time.Second

// transcodeToMP3 converts any ffmpeg-readable input to mono 16 kHz MP3. It is
// the fallback for gateways that reject Telegram's OGG/Opus voice notes.
func transcodeToMP3(ctx context.Context, audio []byte) ([]byte, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, transcodeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffmpeg,
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-i", "pipe:0",
		"-ac", "1", "-ar", "16000",
		"-codec:a", "libmp3lame", "-b:a", "64k",
		"-f", "mp3", "pipe:1",
	)
	cmd.Stdin = bytes.NewReader(audio)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if stdout.Len() == 0 {
		return nil, fmt.Errorf("ffmpeg: empty output")
	}
	return stdout.Bytes(), nil
}
