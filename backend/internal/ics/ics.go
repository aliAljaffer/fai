// Package ics renders all-day events as an RFC 5545 calendar feed.
package ics

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type Event struct {
	UID         string
	Date        time.Time
	Summary     string
	Description string
	URL         string
	Priority    int
	Updated     time.Time
}

// Calendar renders events as a subscribable VCALENDAR. Lines are CRLF-terminated
// and folded at 75 octets, both required by clients like Apple Calendar.
func Calendar(name string, events []Event, now time.Time) string {
	var b strings.Builder

	line(&b, "BEGIN:VCALENDAR")
	line(&b, "VERSION:2.0")
	line(&b, "PRODID:-//fai//fai calendar//EN")
	line(&b, "CALSCALE:GREGORIAN")
	line(&b, "METHOD:PUBLISH")
	line(&b, "X-WR-CALNAME:"+escape(name))
	line(&b, "REFRESH-INTERVAL;VALUE=DURATION:PT30M")
	line(&b, "X-PUBLISHED-TTL:PT30M")

	for _, e := range events {
		line(&b, "BEGIN:VEVENT")
		line(&b, "UID:"+escape(e.UID))
		line(&b, "DTSTAMP:"+now.UTC().Format("20060102T150405Z"))
		line(&b, "DTSTART;VALUE=DATE:"+e.Date.Format("20060102"))
		// DTEND is exclusive for all-day events, so a task due on the 27th
		// occupies only the 27th.
		line(&b, "DTEND;VALUE=DATE:"+e.Date.AddDate(0, 0, 1).Format("20060102"))
		line(&b, "SUMMARY:"+escape(e.Summary))
		if e.Description != "" {
			line(&b, "DESCRIPTION:"+escape(e.Description))
		}
		if e.URL != "" {
			line(&b, "URL:"+escape(e.URL))
		}
		if e.Priority > 0 {
			line(&b, fmt.Sprintf("PRIORITY:%d", e.Priority))
		}
		if !e.Updated.IsZero() {
			line(&b, "LAST-MODIFIED:"+e.Updated.UTC().Format("20060102T150405Z"))
		}
		line(&b, "TRANSP:TRANSPARENT")
		line(&b, "CATEGORIES:fai")
		line(&b, "END:VEVENT")
	}

	line(&b, "END:VCALENDAR")
	return b.String()
}

func line(b *strings.Builder, s string) {
	for i, chunk := range fold(s) {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(chunk)
		b.WriteString("\r\n")
	}
}

// fold splits a content line into chunks that fit in 75 octets, including the
// leading space on continuation lines. It never splits a UTF-8 sequence.
func fold(s string) []string {
	const firstLimit, nextLimit = 75, 74

	var chunks []string
	limit := firstLimit
	for len(s) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		if cut == 0 {
			cut = limit
		}
		chunks = append(chunks, s[:cut])
		s = s[cut:]
		limit = nextLimit
	}
	return append(chunks, s)
}

func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\r\n", `\n`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\n`)
	return s
}
