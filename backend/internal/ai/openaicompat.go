package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const openCodeZenBaseURL = "https://opencode.ai/zen/v1"
const defaultOpenAICompatModel = "gpt-4o-mini"

// openAICompat is a small client for any OpenAI-compatible /chat/completions
// endpoint, configured against OpenCode Zen by default.
type openAICompatProvider struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

func newOpenAICompat(apiKey, model string) Provider {
	if model == "" {
		model = defaultOpenAICompatModel
	}
	return &openAICompatProvider{baseURL: openCodeZenBaseURL, apiKey: apiKey, model: model, http: &http.Client{}}
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (p *openAICompatProvider) Chat(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:    p.model,
		Messages: []chatMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("ai: chat completions returned status %d", resp.StatusCode)
	}

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("ai: no choices in response")
	}
	return parsed.Choices[0].Message.Content, nil
}

func (p *openAICompatProvider) GenerateSubtasks(ctx context.Context, title, descriptionMD string) ([]string, error) {
	prompt := fmt.Sprintf(
		"Break this task into a short list of concrete subtasks. Task: %q. Description: %q.\n"+
			"Respond with ONLY a JSON array of short subtask title strings, nothing else.",
		title, descriptionMD,
	)
	text, err := p.Chat(ctx, prompt)
	if err != nil {
		return nil, err
	}
	return parseJSONStringArray(text)
}
