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

func (p *openAICompatProvider) Chat(ctx context.Context, system string, messages []ChatMessage) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("ai: no messages")
	}
	payload := make([]chatMessage, 0, len(messages)+1)
	if system != "" {
		payload = append(payload, chatMessage{Role: "system", Content: system})
	}
	for _, message := range messages {
		role := "user"
		if message.Role == "assistant" {
			role = "assistant"
		}
		payload = append(payload, chatMessage{Role: role, Content: message.Content})
	}

	body, err := json.Marshal(chatRequest{Model: p.model, Messages: payload})
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
	text, err := p.Chat(ctx, FaiSystem, []ChatMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, err
	}
	return parseJSONStringArray(text)
}

func (p *openAICompatProvider) PlanTask(ctx context.Context, title, body string) (TaskPlan, error) {
	text, err := p.Chat(ctx, FaiSystem, []ChatMessage{{Role: "user", Content: planPrompt(title, body)}})
	if err != nil {
		return TaskPlan{}, err
	}
	return parseTaskPlan(text)
}
