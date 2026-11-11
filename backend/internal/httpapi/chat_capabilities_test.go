package httpapi

import "testing"

func TestChatCapabilitiesCoverEveryAction(t *testing.T) {
	seen := map[string]bool{}
	for _, capability := range chatCapabilities {
		if capability.Summary == "" {
			t.Error("capability without a summary")
		}
		for _, name := range capability.Actions {
			if seen[name] {
				t.Errorf("action %q is listed twice", name)
			}
			seen[name] = true
			if _, ok := chatActionHandlers[name]; !ok {
				t.Errorf("action %q has no handler", name)
			}
		}
	}
	for name := range chatActionHandlers {
		if !seen[name] {
			t.Errorf("action %q is missing from chatCapabilities", name)
		}
	}
	if got := len(ChatCapabilities()); got != len(chatCapabilities) {
		t.Errorf("ChatCapabilities returned %d summaries, want %d", got, len(chatCapabilities))
	}
}
