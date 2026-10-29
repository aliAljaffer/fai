package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

func labelsPrompt(title, descriptionMD string, existing []string) string {
	existingList := "none"
	if len(existing) > 0 {
		existingList = strings.Join(existing, ", ")
	}
	return fmt.Sprintf(
		"Suggest 3 to 6 short keyword labels for this task. Task: %q. Description: %q.\n"+
			"Labels work as tags: lowercase, dash-separated, no spaces, no # prefix.\n"+
			"Existing labels: %s. Reuse an existing label when it fits instead of inventing a synonym.\n"+
			"Respond with ONLY a JSON array of label strings, nothing else.",
		title, truncate(descriptionMD, 4000), existingList,
	)
}

// LabelTask is one task in a batch label request.
type LabelTask struct {
	Title       string
	Description string
}

// LabelsBatchPrompt asks for the labels of every task in one model call.
func LabelsBatchPrompt(tasks []LabelTask, existing []string) string {
	existingList := "none"
	if len(existing) > 0 {
		existingList = strings.Join(existing, ", ")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Suggest 3 to 6 short keyword labels for each of these %d tasks, in order.\n", len(tasks))
	b.WriteString("Labels work as tags: lowercase, dash-separated, no spaces, no # prefix.\n")
	fmt.Fprintf(&b, "Existing labels: %s. Reuse an existing label when it fits instead of inventing a synonym.\n", existingList)
	for i, task := range tasks {
		fmt.Fprintf(&b, "%d. Task: %q. Description: %q\n", i+1, task.Title, truncate(task.Description, 1500))
	}
	b.WriteString("Respond with ONLY a JSON array of arrays: one inner array of label strings per task, nothing else.")
	return b.String()
}

// ParseLabelBatches returns one label list per task, padding missing tasks with
// an empty list so callers can index results by task order.
func ParseLabelBatches(text string, want int) ([][]string, error) {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("ai: no JSON array found in response")
	}
	var out [][]string
	if err := json.Unmarshal([]byte(text[start:end+1]), &out); err != nil {
		return nil, fmt.Errorf("ai: invalid JSON array in response: %w", err)
	}
	if len(out) > want {
		out = out[:want]
	}
	for len(out) < want {
		out = append(out, nil)
	}
	return out, nil
}
