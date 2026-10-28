package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

func planPrompt(title, body string) string {
	return fmt.Sprintf(
		"Turn this note into one actionable task and a short list of concrete subtasks.\n"+
			"Note title: %q\nNote:\n%s\n\n"+
			`Respond with ONLY JSON in this shape: {"title": string, "subtasks": [string]}. `+
			"Use 2 to 8 subtasks. Keep each title short. Do not add commentary.",
		title, truncate(body, 6000),
	)
}

func parseTaskPlan(text string) (TaskPlan, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end < start {
		return TaskPlan{}, fmt.Errorf("ai: no JSON object found in response")
	}
	var plan TaskPlan
	if err := json.Unmarshal([]byte(text[start:end+1]), &plan); err != nil {
		return TaskPlan{}, fmt.Errorf("ai: invalid JSON object in response: %w", err)
	}
	plan.Title = strings.TrimSpace(plan.Title)
	if plan.Title == "" {
		return TaskPlan{}, fmt.Errorf("ai: response has no task title")
	}
	seen := make(map[string]struct{}, len(plan.Subtasks))
	out := make([]string, 0, len(plan.Subtasks))
	for _, subtask := range plan.Subtasks {
		title := strings.TrimSpace(subtask)
		if title == "" {
			continue
		}
		key := strings.ToLower(title)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, title)
		if len(out) == 12 {
			break
		}
	}
	if len(out) == 0 {
		return TaskPlan{}, fmt.Errorf("ai: response has no subtasks")
	}
	plan.Subtasks = out
	return plan, nil
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
