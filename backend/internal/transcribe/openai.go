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
	defaultOpenAIModel = "whisper-1"
	openAITimeout      = 120 * time.Second
)

type OpenAICompat struct {
	baseURL  string
	model    string
	apiKey   string
	language string
	http     *http.Client
}

func NewOpenAICompat(baseURL, model, apiKey, language string) *OpenAICompat {
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultOpenAIModel
	}
	return &OpenAICompat{
		baseURL:  strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		model:    model,
		apiKey:   strings.TrimSpace(apiKey),
		language: strings.TrimSpace(language),
		http:     &http.Client{Timeout: openAITimeout},
	}
}

func (o *OpenAICompat) Transcribe(ctx context.Context, audio []byte, format string) (string, error) {
	if len(audio) == 0 {
		return "", fmt.Errorf("transcribe: no audio")
	}
	if o.baseURL == "" {
		return "", fmt.Errorf("transcribe: no base URL")
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
	if err := writer.WriteField("model", o.model); err != nil {
		return "", err
	}
	if o.language != "" {
		if err := writer.WriteField("language", o.language); err != nil {
			return "", err
		}
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/audio/transcriptions", &body)
	if err != nil {
		return "", err
	}
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := o.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("transcribe: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("transcribe: status %d: %s", resp.StatusCode, openAIErrorMessage(payload))
	}

	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("transcribe: %w", err)
	}
	return strings.TrimSpace(parsed.Text), nil
}

func openAIErrorMessage(body []byte) string {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) == nil && envelope.Error.Message != "" {
		return envelope.Error.Message
	}
	text := strings.TrimSpace(string(body))
	if len(text) > 300 {
		text = text[:300]
	}
	return text
}
