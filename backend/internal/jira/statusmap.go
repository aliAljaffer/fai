package jira

import "strings"

// ParseStatusColumnMap parses "Jira Status:Column Name,..." into a lookup map.
// Format: comma-separated pairs, status and column name separated by a colon.
func ParseStatusColumnMap(raw string) map[string]string {
	m := map[string]string{}
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}
		m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return m
}
