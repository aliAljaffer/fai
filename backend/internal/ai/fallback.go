package ai

import (
	"context"
	"errors"
	"log"
)

type fallbackProvider struct {
	primary  Provider
	fallback Provider
}

func WithFallback(primary, fallback Provider) Provider {
	if primary == nil {
		return fallback
	}
	if fallback == nil {
		return primary
	}
	return &fallbackProvider{primary: primary, fallback: fallback}
}

func (f *fallbackProvider) Chat(ctx context.Context, system string, messages []ChatMessage) (string, error) {
	out, err := f.primary.Chat(ctx, system, messages)
	if err == nil {
		return out, nil
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	log.Printf("ai: primary model failed, trying fallback: %v", err)
	return f.fallback.Chat(ctx, system, messages)
}

func (f *fallbackProvider) GenerateSubtasks(ctx context.Context, title, descriptionMD string) ([]string, error) {
	out, err := f.primary.GenerateSubtasks(ctx, title, descriptionMD)
	if err == nil {
		return out, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	log.Printf("ai: primary model failed, trying fallback: %v", err)
	return f.fallback.GenerateSubtasks(ctx, title, descriptionMD)
}

func (f *fallbackProvider) SuggestLabels(ctx context.Context, title, descriptionMD string, existing []string) ([]string, error) {
	out, err := f.primary.SuggestLabels(ctx, title, descriptionMD, existing)
	if err == nil {
		return out, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	log.Printf("ai: primary model failed, trying fallback: %v", err)
	return f.fallback.SuggestLabels(ctx, title, descriptionMD, existing)
}

func (f *fallbackProvider) PlanTask(ctx context.Context, title, body string) (TaskPlan, error) {
	out, err := f.primary.PlanTask(ctx, title, body)
	if err == nil {
		return out, nil
	}
	if ctx.Err() != nil {
		return TaskPlan{}, ctx.Err()
	}
	log.Printf("ai: primary model failed, trying fallback: %v", err)
	return f.fallback.PlanTask(ctx, title, body)
}

func (f *fallbackProvider) Transcribe(ctx context.Context, audio []byte, format string) (string, error) {
	out, err := f.primary.Transcribe(ctx, audio, format)
	if err == nil {
		return out, nil
	}
	if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "", err
	}
	log.Printf("ai: primary model failed, trying fallback: %v", err)
	return f.fallback.Transcribe(ctx, audio, format)
}
