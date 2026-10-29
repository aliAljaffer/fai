package ai

import (
	"strings"
	"testing"
)

func TestStepSystemKeepsStepRules(t *testing.T) {
	for _, rule := range []string{"You are Fai", "ASD-STE100", "imperative", "one action per step", "under 20 words"} {
		if !strings.Contains(StepSystem, rule) {
			t.Errorf("StepSystem is missing %q", rule)
		}
	}
}
