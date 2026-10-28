package ai

import (
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
