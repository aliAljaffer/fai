package store

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	workspaceTagPattern = regexp.MustCompile(`^[A-Za-z]{3,4}$`)
	bodyLabelPattern    = regexp.MustCompile(`#([A-Za-z0-9][A-Za-z0-9-]*)`)
	taskRefPattern      = regexp.MustCompile(`^([A-Za-z]{3,4})-([0-9]{1,18})$`)
)

// ValidWorkspaceTag reports whether tag is 3 or 4 ASCII letters.
func ValidWorkspaceTag(tag string) bool {
	return workspaceTagPattern.MatchString(tag)
}

// ParseBodyLabels returns the distinct #label tokens in body, in first-seen order.
// A space ends a token; the editor inserts a dash instead while typing a label.
func ParseBodyLabels(body string) []string {
	var out []string
	seen := map[string]bool{}
	for _, match := range bodyLabelPattern.FindAllStringSubmatch(body, -1) {
		label := strings.Trim(match[1], "-")
		if label == "" || seen[strings.ToLower(label)] {
			continue
		}
		seen[strings.ToLower(label)] = true
		out = append(out, label)
	}
	return out
}

// EscapeLike escapes a literal for use in a LIKE pattern with ESCAPE '\'.
func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	return strings.ReplaceAll(s, "_", `\_`)
}

// ParseTaskRef splits "FAI-42" into its tag and number.
func ParseTaskRef(q string) (tag string, num int64, ok bool) {
	match := taskRefPattern.FindStringSubmatch(strings.TrimSpace(q))
	if match == nil {
		return "", 0, false
	}
	n, err := strconv.ParseInt(match[2], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return strings.ToUpper(match[1]), n, true
}
