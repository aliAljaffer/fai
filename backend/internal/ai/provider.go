package ai

import "context"

type Provider interface {
	GenerateSubtasks(ctx context.Context, title, descriptionMD string) ([]string, error)
	Chat(ctx context.Context, prompt string) (string, error)
}

// NewProvider returns nil if no AI_API_KEY is configured, so every AI endpoint
// can degrade gracefully (503, hidden in the UI) rather than treating AI as a
// hard dependency.
func NewProvider(provider, apiKey, model string) Provider {
	if apiKey == "" {
		return nil
	}
	switch provider {
	case "openai-compat", "opencode-zen":
		return newOpenAICompat(apiKey, model)
	default:
		return newAnthropic(apiKey, model)
	}
}
