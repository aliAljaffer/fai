package ics

import (
	"testing"
	"time"
)

func TestParseAllDayEvent(t *testing.T) {
	data := []byte("BEGIN:VCALENDAR\r\n" +
		"VERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:abc-1\r\n" +
		"SUMMARY:Team offsite\\, all day\r\n" +
		"DTSTART;VALUE=DATE:20261027\r\n" +
		"DTEND;VALUE=DATE:20261028\r\n" +
		"LOCATION:Office\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n")

	got, err := Parse(data, day(2026, 10, 1), day(2026, 11, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d events", len(got))
	}
	event := got[0]
	if event.Summary != "Team offsite, all day" {
		t.Errorf("summary = %q", event.Summary)
	}
	if !event.AllDay || event.Start.Format("2006-01-02") != "2026-10-27" {
		t.Errorf("wrong start: %#v", event)
	}
	if event.End.Format("2006-01-02") != "2026-10-28" {
		t.Errorf("wrong end: %#v", event)
	}
	if event.Location != "Office" {
		t.Errorf("location = %q", event.Location)
	}
}

func TestParseTimedEventWithFoldedLine(t *testing.T) {
	data := []byte("BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:timed-1\r\n" +
		"SUMMARY:A very long title that is folded across\r\n" +
		"  two physical lines\r\n" +
		"DTSTART:20261027T093000Z\r\n" +
		"DTEND:20261027T103000Z\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n")

	got, err := Parse(data, day(2026, 10, 1), day(2026, 11, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d events", len(got))
	}
	if got[0].Summary != "A very long title that is folded across two physical lines" {
		t.Errorf("summary = %q", got[0].Summary)
	}
	if got[0].Start.Hour() != 9 || got[0].AllDay {
		t.Errorf("wrong start: %#v", got[0])
	}
}

func TestParseRecurringEventExpandsWithinWindow(t *testing.T) {
	data := []byte("BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:weekly-1\r\n" +
		"SUMMARY:Weekly review\r\n" +
		"DTSTART;VALUE=DATE:20261005\r\n" +
		"DTEND;VALUE=DATE:20261006\r\n" +
		"RRULE:FREQ=WEEKLY;COUNT=4\r\n" +
		"EXDATE;VALUE=DATE:20261012\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n")

	got, err := Parse(data, day(2026, 10, 1), day(2026, 11, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d occurrences, want 3", len(got))
	}
	want := []string{"2026-10-05", "2026-10-19", "2026-10-26"}
	for i, event := range got {
		if gotDate := event.Start.Format("2006-01-02"); gotDate != want[i] {
			t.Errorf("occurrence %d = %s, want %s", i, gotDate, want[i])
		}
	}
}

func TestParseSkipsAlarms(t *testing.T) {
	data := []byte("BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:alarm-1\r\n" +
		"SUMMARY:Dentist\r\n" +
		"DTSTART:20261027T120000Z\r\n" +
		"BEGIN:VALARM\r\n" +
		"SUMMARY:Reminder\r\n" +
		"TRIGGER:-PT30M\r\n" +
		"END:VALARM\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n")

	got, err := Parse(data, day(2026, 10, 1), day(2026, 11, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Summary != "Dentist" {
		t.Fatalf("alarm summary leaked: %#v", got)
	}
}

func TestParseIncludesOverlappingMultiDayEvent(t *testing.T) {
	data := []byte("BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:long-1\r\n" +
		"SUMMARY:Conference\r\n" +
		"DTSTART;VALUE=DATE:20260928\r\n" +
		"DTEND;VALUE=DATE:20261003\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n")

	got, err := Parse(data, day(2026, 10, 1), day(2026, 11, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
}

func day(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
