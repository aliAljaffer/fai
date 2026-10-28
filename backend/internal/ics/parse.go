package ics

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/teambition/rrule-go"
)

// Occurrence is one dated instance parsed from an external ICS feed. Recurring
// events are expanded within the requested window.
type Occurrence struct {
	UID         string
	Summary     string
	Description string
	Location    string
	URL         string
	Start       time.Time
	End         time.Time
	AllDay      bool
}

type property struct {
	params map[string]string
	value  string
}

const (
	maxCalendarBytes = 8 << 20
	maxOccurrences   = 1000
)

// Parse reads an ICS calendar and returns the events that overlap [from, to).
func Parse(data []byte, from, to time.Time) ([]Occurrence, error) {
	lines, err := unfold(data)
	if err != nil {
		return nil, err
	}

	var out []Occurrence
	var current map[string][]property
	inEvent := false
	nested := 0

	for _, line := range lines {
		if line == "" {
			continue
		}
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "BEGIN:"):
			component := strings.ToUpper(strings.TrimSpace(line[len("BEGIN:"):]))
			if component == "VEVENT" && !inEvent {
				inEvent = true
				nested = 0
				current = make(map[string][]property)
				continue
			}
			if inEvent {
				nested++
			}
		case strings.HasPrefix(upper, "END:"):
			component := strings.ToUpper(strings.TrimSpace(line[len("END:"):]))
			if !inEvent {
				continue
			}
			if component == "VEVENT" && nested == 0 {
				out = append(out, expandEvent(current, from, to)...)
				inEvent = false
				current = nil
				continue
			}
			if nested > 0 {
				nested--
			}
		case inEvent && nested == 0:
			name, prop, ok := parseLine(line)
			if ok {
				current[name] = append(current[name], prop)
			}
		}
	}

	return out, nil
}

func unfold(data []byte) ([]string, error) {
	if len(data) > maxCalendarBytes {
		return nil, fmt.Errorf("calendar is larger than %d bytes", maxCalendarBytes)
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), maxCalendarBytes)

	var lines []string
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(lines) > 0 {
			lines[len(lines)-1] += line[1:]
			continue
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}

func parseLine(line string) (string, property, bool) {
	colon := strings.Index(line, ":")
	if colon < 0 {
		return "", property{}, false
	}
	left, value := line[:colon], line[colon+1:]
	parts := strings.Split(left, ";")
	name := strings.ToUpper(strings.TrimSpace(parts[0]))
	params := make(map[string]string, len(parts)-1)
	for _, part := range parts[1:] {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		params[strings.ToUpper(strings.TrimSpace(kv[0]))] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
	}
	return name, property{params: params, value: value}, true
}

func expandEvent(props map[string][]property, from, to time.Time) []Occurrence {
	startProp, ok := firstProp(props, "DTSTART")
	if !ok {
		return nil
	}
	start, allDay, err := parseDate(startProp.value, startProp.params)
	if err != nil {
		return nil
	}

	duration := defaultDuration(allDay)
	if endProp, ok := firstProp(props, "DTEND"); ok {
		if end, _, err := parseDate(endProp.value, endProp.params); err == nil && !end.Before(start) {
			duration = end.Sub(start)
		}
	} else if d, ok := firstProp(props, "DURATION"); ok {
		if parsed, err := parseDuration(d.value); err == nil {
			duration = parsed
		}
	}

	summary := "Untitled event"
	if p, ok := firstProp(props, "SUMMARY"); ok && strings.TrimSpace(p.value) != "" {
		summary = unescapeText(p.value)
	}
	uid := strconv.FormatInt(start.Unix(), 10) + "-" + summary
	if p, ok := firstProp(props, "UID"); ok && strings.TrimSpace(p.value) != "" {
		uid = unescapeText(p.value)
	}
	description := textProp(props, "DESCRIPTION")
	location := textProp(props, "LOCATION")
	link := textProp(props, "URL")

	starts := []time.Time{start}
	if ruleProp, ok := firstProp(props, "RRULE"); ok {
		starts = expandRecurrence(ruleProp.value, props, start, duration, from, to)
	}

	seen := make(map[int64]struct{}, len(starts))
	out := make([]Occurrence, 0, len(starts))
	for _, occurrence := range starts {
		if _, duplicate := seen[occurrence.UnixNano()]; duplicate {
			continue
		}
		seen[occurrence.UnixNano()] = struct{}{}
		end := occurrence.Add(duration)
		if !end.After(from) || !occurrence.Before(to) {
			continue
		}
		out = append(out, Occurrence{
			UID: uid, Summary: summary, Description: description, Location: location,
			URL: link, Start: occurrence, End: end, AllDay: allDay,
		})
	}
	return out
}

func expandRecurrence(ruleValue string, props map[string][]property, start time.Time, duration time.Duration, from, to time.Time) []time.Time {
	rule, err := rrule.StrToRRule(ruleValue)
	if err != nil {
		return nil
	}
	rule.DTStart(start)
	set := rrule.Set{}
	set.DTStart(start)
	set.RRule(rule)

	for _, exdate := range props["EXDATE"] {
		for _, value := range strings.Split(exdate.value, ",") {
			if t, _, err := parseDate(value, exdate.params); err == nil {
				set.ExDate(t)
			}
		}
	}
	for _, rdate := range props["RDATE"] {
		for _, value := range strings.Split(rdate.value, ",") {
			if t, _, err := parseDate(value, rdate.params); err == nil {
				set.RDate(t)
			}
		}
	}

	starts := set.Between(from.Add(-duration), to, true)
	if len(starts) > maxOccurrences {
		starts = starts[:maxOccurrences]
	}
	return starts
}

func firstProp(props map[string][]property, name string) (property, bool) {
	values, ok := props[name]
	if !ok || len(values) == 0 {
		return property{}, false
	}
	return values[0], true
}

func textProp(props map[string][]property, name string) string {
	p, ok := firstProp(props, name)
	if !ok {
		return ""
	}
	return unescapeText(p.value)
}

func defaultDuration(allDay bool) time.Duration {
	if allDay {
		return 24 * time.Hour
	}
	return 0
}

func parseDate(value string, params map[string]string) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false, fmt.Errorf("empty date")
	}

	loc := time.Local
	if tzid := params["TZID"]; tzid != "" {
		if loaded, err := time.LoadLocation(normalizeTZID(tzid)); err == nil {
			loc = loaded
		} else {
			loc = time.UTC
		}
	}

	if len(value) == 8 && !strings.Contains(value, "T") {
		t, err := time.ParseInLocation("20060102", value, time.UTC)
		return t, true, err
	}
	if strings.HasSuffix(value, "Z") {
		for _, layout := range []string{"20060102T150405Z", "20060102T1504Z"} {
			if t, err := time.Parse(layout, value); err == nil {
				return t, false, nil
			}
		}
	}
	for _, layout := range []string{"20060102T150405", "20060102T1504"} {
		if t, err := time.ParseInLocation(layout, value, loc); err == nil {
			return t, false, nil
		}
	}
	return time.Time{}, false, fmt.Errorf("unsupported date %q", value)
}

func normalizeTZID(tzid string) string {
	tzid = strings.Trim(tzid, `"`)
	if strings.HasPrefix(tzid, "/") {
		parts := strings.Split(strings.TrimPrefix(tzid, "/"), "/")
		if len(parts) >= 2 {
			return strings.Join(parts[len(parts)-2:], "/")
		}
	}
	return tzid
}

var durationPattern = regexp.MustCompile(`^P(?:(\d+)W)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

func parseDuration(value string) (time.Duration, error) {
	matches := durationPattern.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return 0, fmt.Errorf("unsupported duration %q", value)
	}
	units := []time.Duration{time.Hour * 24 * 7, time.Hour * 24, time.Hour, time.Minute, time.Second}
	var total time.Duration
	for i, raw := range matches[1:] {
		if raw == "" {
			continue
		}
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, err
		}
		total += time.Duration(n) * units[i]
	}
	return total, nil
}

func unescapeText(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n', 'N':
			b.WriteByte('\n')
		case ',', ';', '\\':
			b.WriteByte(s[i])
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
