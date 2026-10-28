package ai

import "context"

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

type Provider interface {
	Chat(ctx context.Context, system string, messages []ChatMessage) (string, error)
	GenerateSubtasks(ctx context.Context, title, descriptionMD string) ([]string, error)
	SuggestLabels(ctx context.Context, title, descriptionMD string, existing []string) ([]string, error)
	PlanTask(ctx context.Context, title, body string) (TaskPlan, error)
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
