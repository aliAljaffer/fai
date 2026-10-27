package ics

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestCalendarShape(t *testing.T) {
	date := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 10, 26, 18, 0, 0, 0, time.UTC)
	out := Calendar("fai tasks", []Event{{
		UID:         "task-7@fai",
		Date:        date,
		Summary:     "Buy milk",
		Description: "Priority: high",
		URL:         "https://fai.example/?task=7",
		Priority:    1,
		Updated:     updated,
	}}, time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC))

	for _, want := range []string{
		"BEGIN:VCALENDAR\r\n",
		"DTSTART;VALUE=DATE:20261027\r\n",
		"DTEND;VALUE=DATE:20261028\r\n",
		"SUMMARY:Buy milk\r\n",
		"PRIORITY:1\r\n",
		"DTSTAMP:20261027T120000Z\r\n",
		"LAST-MODIFIED:20261026T180000Z\r\n",
		"URL:https://fai.example/?task=7\r\n",
		"END:VCALENDAR\r\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
	if strings.HasSuffix(out, "\n\n") {
		t.Error("trailing blank line")
	}
}

func TestCalendarEscapesText(t *testing.T) {
	out := Calendar("x", []Event{{
		UID:     "u",
		Date:    time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Summary: `a,b;c\d` + "\n" + "next",
	}}, time.Now())
	if !strings.Contains(out, `SUMMARY:a\,b\;c\\d\nnext`) {
		t.Errorf("text was not escaped:\n%s", out)
	}
}

func TestFoldKeepsLinesWithinLimitAndRunesIntact(t *testing.T) {
	long := strings.Repeat("hello, ", 40) + strings.Repeat("مرحبا ", 30)
	out := Calendar("x", []Event{{
		UID:     "u",
		Date:    time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Summary: long,
	}}, time.Now())

	for _, physical := range strings.Split(strings.TrimSuffix(out, "\r\n"), "\r\n") {
		if len(physical) > 75 {
			t.Errorf("line is %d octets: %q", len(physical), physical)
		}
		if !utf8.ValidString(physical) {
			t.Errorf("fold split a UTF-8 sequence: %q", physical)
		}
	}

	folded := false
	for _, physical := range strings.Split(out, "\r\n") {
		if strings.HasPrefix(physical, " ") {
			folded = true
			break
		}
	}
	if !folded {
		t.Error("long line was not folded")
	}
	joined := strings.ReplaceAll(strings.Split(out, "SUMMARY:")[1], "\r\n ", "")
	if !strings.HasPrefix(joined, escape(long)) {
		t.Error("unfolding did not restore the original text")
	}
}
