package recurrence

import (
	"errors"
	"testing"
	"time"
)

func at(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func TestNext(t *testing.T) {
	tests := []struct {
		name   string
		rule   Rule
		anchor string
		after  string
		want   string
	}{
		{"daily next day", Rule{Daily, 1}, "2026-01-01T09:00:00Z", "2026-01-01T09:00:00Z", "2026-01-02T09:00:00Z"},
		{"daily strictly after an occurrence", Rule{Daily, 1}, "2026-01-01T09:00:00Z", "2026-01-05T09:00:00Z", "2026-01-06T09:00:00Z"},
		{"daily between occurrences", Rule{Daily, 3}, "2026-01-01T09:00:00Z", "2026-01-05T10:00:00Z", "2026-01-07T09:00:00Z"},
		{"after before the anchor returns the anchor", Rule{Daily, 2}, "2026-01-10T09:00:00Z", "2026-01-01T00:00:00Z", "2026-01-10T09:00:00Z"},
		{"late completion skips missed occurrences", Rule{Daily, 1}, "2026-01-01T09:00:00Z", "2027-03-15T12:00:00Z", "2027-03-16T09:00:00Z"},
		{"weekly interval 2", Rule{Weekly, 2}, "2026-01-05T08:30:00Z", "2026-01-06T00:00:00Z", "2026-01-19T08:30:00Z"},
		{"weekly keeps the weekday across years", Rule{Weekly, 1}, "2026-01-05T08:30:00Z", "2029-06-01T00:00:00Z", "2029-06-04T08:30:00Z"},
		{"monthly clamps to February", Rule{Monthly, 1}, "2026-01-31T10:00:00Z", "2026-01-31T10:00:00Z", "2026-02-28T10:00:00Z"},
		{"monthly returns to the 31st", Rule{Monthly, 1}, "2026-01-31T10:00:00Z", "2026-02-28T10:00:00Z", "2026-03-31T10:00:00Z"},
		{"monthly clamps to April 30", Rule{Monthly, 1}, "2026-01-31T10:00:00Z", "2026-03-31T10:00:00Z", "2026-04-30T10:00:00Z"},
		{"monthly clamps to leap February", Rule{Monthly, 1}, "2028-01-31T10:00:00Z", "2028-01-31T10:00:00Z", "2028-02-29T10:00:00Z"},
		{"monthly interval 3", Rule{Monthly, 3}, "2026-01-15T10:00:00Z", "2026-01-16T00:00:00Z", "2026-04-15T10:00:00Z"},
		{"monthly crosses the year", Rule{Monthly, 1}, "2026-11-30T10:00:00Z", "2026-12-31T00:00:00Z", "2027-01-30T10:00:00Z"},
		{"monthly far in the future", Rule{Monthly, 1}, "2026-01-31T10:00:00Z", "2031-07-01T00:00:00Z", "2031-07-31T10:00:00Z"},
		{"yearly leap day clamps", Rule{Yearly, 1}, "2024-02-29T10:00:00Z", "2024-02-29T10:00:00Z", "2025-02-28T10:00:00Z"},
		{"yearly leap day returns", Rule{Yearly, 1}, "2024-02-29T10:00:00Z", "2027-03-01T00:00:00Z", "2028-02-29T10:00:00Z"},
		{"yearly interval 2", Rule{Yearly, 2}, "2026-05-01T10:00:00Z", "2026-05-01T10:00:00Z", "2028-05-01T10:00:00Z"},
		{"offsets are read as UTC", Rule{Daily, 1}, "2026-01-01T23:00:00-05:00", "2026-01-02T03:00:00Z", "2026-01-02T04:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.Next(at(tt.anchor), at(tt.after))
			if want := at(tt.want); !got.Equal(want) {
				t.Errorf("Next = %s, want %s", got.Format(time.RFC3339), tt.want)
			}
		})
	}
}

func TestNextIsAlwaysAfterAndOnTheSeries(t *testing.T) {
	anchor := at("2026-01-31T10:00:00Z")
	for _, rule := range []Rule{{Daily, 1}, {Daily, 7}, {Weekly, 2}, {Monthly, 1}, {Monthly, 5}, {Yearly, 1}} {
		after := anchor.Add(-48 * time.Hour)
		for range 80 {
			next := rule.Next(anchor, after)
			if !next.After(after) {
				t.Fatalf("%v: Next(%s) = %s is not after", rule, after, next)
			}
			after = next.Add(-time.Hour)
			if again := rule.Next(anchor, after); !again.Equal(next) {
				t.Fatalf("%v: Next is not stable: %s then %s", rule, next, again)
			}
			after = next
		}
	}
}

func TestNextDate(t *testing.T) {
	got, err := Rule{Monthly, 1}.NextDate("2026-01-31", "2026-02-28")
	if err != nil || got != "2026-03-31" {
		t.Errorf("NextDate = %q, %v", got, err)
	}
	if _, err := (Rule{Daily, 1}).NextDate("nope", "2026-01-01"); err == nil {
		t.Error("expected an error for a bad anchor")
	}
	if _, err := (Rule{Daily, 1}).NextDate("2026-01-01", "nope"); err == nil {
		t.Error("expected an error for a bad date")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		rule Rule
		ok   bool
	}{
		{Rule{Daily, 1}, true},
		{Rule{Yearly, MaxInterval}, true},
		{Rule{"", 1}, false},
		{Rule{"hourly", 1}, false},
		{Rule{Daily, 0}, false},
		{Rule{Daily, -1}, false},
		{Rule{Daily, MaxInterval + 1}, false},
	}
	for _, tt := range tests {
		err := tt.rule.Validate()
		if (err == nil) != tt.ok {
			t.Errorf("Validate(%v) = %v, want ok=%v", tt.rule, err, tt.ok)
		}
		if err != nil && !errors.Is(err, ErrInvalid) {
			t.Errorf("Validate(%v) error does not wrap ErrInvalid", tt.rule)
		}
	}
}

func TestDescribe(t *testing.T) {
	tests := map[Rule]string{
		{Daily, 1}:   "every day",
		{Weekly, 2}:  "every 2 weeks",
		{Monthly, 1}: "every month",
		{Yearly, 3}:  "every 3 years",
	}
	for rule, want := range tests {
		if got := rule.Describe(); got != want {
			t.Errorf("Describe(%v) = %q, want %q", rule, got, want)
		}
	}
}
