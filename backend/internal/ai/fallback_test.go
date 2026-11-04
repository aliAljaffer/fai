package ai

import (
	"context"
	"errors"
	"testing"
)

type stubProvider struct {
	out  string
	err  error
	used *[]string
}

func (s stubProvider) record() {
	if s.used != nil {
		*s.used = append(*s.used, s.out)
	}
}

func (s stubProvider) Chat(context.Context, string, []ChatMessage) (string, error) {
	s.record()
	return s.out, s.err
}

func (s stubProvider) GenerateSubtasks(context.Context, string, string) ([]string, error) {
	s.record()
	return []string{s.out}, s.err
}

func (s stubProvider) SuggestLabels(context.Context, string, string, []string) ([]string, error) {
	s.record()
	return []string{s.out}, s.err
}

func (s stubProvider) PlanTask(context.Context, string, string) (TaskPlan, error) {
	s.record()
	return TaskPlan{Title: s.out}, s.err
}

func (s stubProvider) Transcribe(context.Context, []byte, string) (string, error) {
	s.record()
	return s.out, s.err
}

func TestFallbackUsesSecondaryOnPrimaryError(t *testing.T) {
	var used []string
	provider := WithFallback(
		stubProvider{out: "primary", err: errors.New("rate limited"), used: &used},
		stubProvider{out: "fallback", used: &used},
	)
	text, err := provider.Chat(context.Background(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if text != "fallback" {
		t.Fatalf("text = %q", text)
	}
	if len(used) != 2 || used[0] != "primary" || used[1] != "fallback" {
		t.Fatalf("used = %#v", used)
	}
}
