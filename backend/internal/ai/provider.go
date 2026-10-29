package ai

import (
	"context"
	"strings"
)

// ChatMessage is one conversation turn. Only user and assistant roles are
// accepted from clients; the system instruction is chosen by the server.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TaskPlan struct {
	Title    string   `json:"title"`
	Subtasks []string `json:"subtasks"`
}

// FaiSystem is the fixed instruction for every Fai conversation.
const FaiSystem = "You are Fai, the assistant built into a personal task, note, and calendar app. " +
	"Help the owner turn ideas, notes, and meetings into small concrete tasks and subtasks. " +
	"Be direct and practical. Use only the data the app gives you and say when you do not know. " +
	"Do not invent tasks, dates, or facts. Do not use em dashes. Keep replies short."

const (
	ProviderAnthropic    = "anthropic"
	ProviderOpenAI       = "openai"
	ProviderOpenAICompat = "openai-compat"
	ProviderOpenCodeGo   = "opencode-go"
)

// NormalizeProvider maps known aliases to the stored provider name. Unknown
// values fall back to Anthropic, which keeps an unset AI_PROVIDER working.
func NormalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case ProviderOpenAI:
		return ProviderOpenAI
	case ProviderOpenAICompat, "opencode-zen", "opencode":
		return ProviderOpenAICompat
	case ProviderOpenCodeGo:
		return ProviderOpenCodeGo
	default:
		return ProviderAnthropic
	}
}

type Provider interface {
	Chat(ctx context.Context, system string, messages []ChatMessage) (string, error)
	GenerateSubtasks(ctx context.Context, title, descriptionMD string) ([]string, error)
	SuggestLabels(ctx context.Context, title, descriptionMD string, existing []string) ([]string, error)
	PlanTask(ctx context.Context, title, body string) (TaskPlan, error)
	Transcribe(ctx context.Context, audio []byte, format string) (string, error)
}

// NewProvider returns nil if no API key is configured, so every AI endpoint
// can degrade gracefully (503, hidden in the UI) rather than treating AI as a
// hard dependency.
func NewProvider(provider, baseURL, apiKey, model string) Provider {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	switch NormalizeProvider(provider) {
	case ProviderOpenAI:
		return newOpenAICompat(baseURL, openAIBaseURL, apiKey, model)
	case ProviderOpenAICompat:
		return newOpenAICompat(baseURL, openCodeZenBaseURL, apiKey, model)
	case ProviderOpenCodeGo:
		return newOpenAICompat(baseURL, openCodeGoBaseURL, apiKey, model)
	default:
		return newAnthropic(baseURL, apiKey, model)
	}
}
