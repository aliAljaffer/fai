package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const defaultAnthropicModel = "claude-sonnet-5"

type anthropicProvider struct {
	client anthropic.Client
	model  string
}

func newAnthropic(apiKey, model string) Provider {
	if model == "" {
		model = defaultAnthropicModel
	}
	return &anthropicProvider{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

func (p *anthropicProvider) Chat(ctx context.Context, prompt string) (string, error) {
	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: 1024,
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
	})
	if err != nil {
		return "", err
	}
	return textFromBlocks(msg.Content), nil
}

func (p *anthropicProvider) GenerateSubtasks(ctx context.Context, title, descriptionMD string) ([]string, error) {
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

func textFromBlocks(blocks []anthropic.ContentBlockUnion) string {
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}

// parseJSONStringArray extracts a JSON array of strings from model output, tolerating
// surrounding prose or a markdown code fence since models don't always follow
// "respond with only JSON" instructions exactly.
func parseJSONStringArray(text string) ([]string, error) {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("ai: no JSON array found in response")
	}
	var out []string
	if err := json.Unmarshal([]byte(text[start:end+1]), &out); err != nil {
		return nil, fmt.Errorf("ai: invalid JSON array in response: %w", err)
	}
	return out, nil
}
