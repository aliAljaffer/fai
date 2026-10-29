package transcribe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.elevenlabs.io"
	defaultModel   = "scribe_v2"
	requestTimeout = 120 * time.Second
)

var contentTypes = map[string]string{
	"mp3": "audio/mpeg",
	"wav": "audio/wav",
	"m4a": "audio/mp4",
	"ogg": "audio/ogg",
}

type ElevenLabs struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

func NewElevenLabs(apiKey, model string) *ElevenLabs {
	return newElevenLabs(apiKey, model, defaultBaseURL)
}

func newElevenLabs(apiKey, model, baseURL string) *ElevenLabs {
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModel
	}
	return &ElevenLabs{
		apiKey:  strings.TrimSpace(apiKey),
		model:   model,
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		http:    &http.Client{Timeout: requestTimeout},
	}
}

func (e *ElevenLabs) Transcribe(ctx context.Context, audio []byte, format string) (string, error) {
	if len(audio) == 0 {
		return "", fmt.Errorf("transcribe: no audio")
	}
	if e.apiKey == "" {
		return "", fmt.Errorf("transcribe: no API key")
	}

	format = normalizeFormat(format)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="voice.%s"`, format))
	header.Set("Content-Type", contentTypes[format])
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(audio); err != nil {
		return "", err
	}
	if err := writer.WriteField("model_id", e.model); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/v1/speech-to-text", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("xi-api-key", e.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := e.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("transcribe: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("transcribe: status %d: %s", resp.StatusCode, responseMessage(payload))
	}

	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("transcribe: %w", err)
	}
	return strings.TrimSpace(parsed.Text), nil
}

func normalizeFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "mp3", "wav", "m4a":
		return format
	default:
		return "ogg"
	}
}

func responseMessage(body []byte) string {
	var envelope struct {
		Detail json.RawMessage `json:"detail"`
	}
	if json.Unmarshal(body, &envelope) == nil && len(envelope.Detail) > 0 {
		var detail struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(envelope.Detail, &detail) == nil && detail.Message != "" {
			return detail.Message
		}
	}
	text := strings.TrimSpace(string(body))
	if len(text) > 300 {
		text = text[:300]
	}
	return text
}
