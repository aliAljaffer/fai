// Package recurrence holds the repeat rule shared by reminders and tasks and the
// maths for finding the next occurrence. It has no storage or HTTP code.
package recurrence

import (
	"errors"
	"fmt"
	"time"
)

type Freq string

const (
	Daily   Freq = "daily"
	Weekly  Freq = "weekly"
	Monthly Freq = "monthly"
	Yearly  Freq = "yearly"
)

const MaxInterval = 365

const DateLayout = "2006-01-02"

var ErrInvalid = errors.New("invalid repeat rule")

type Rule struct {
	Freq     Freq `json:"freq"`
	Interval int  `json:"interval"`
}

func (r Rule) Validate() error {
	switch r.Freq {
	case Daily, Weekly, Monthly, Yearly:
	default:
		return fmt.Errorf("%w: freq must be daily, weekly, monthly or yearly", ErrInvalid)
	}
	if r.Interval < 1 || r.Interval > MaxInterval {
		return fmt.Errorf("%w: interval must be between 1 and %d", ErrInvalid, MaxInterval)
	}
	return nil
}

func (r Rule) Describe() string {
	unit := map[Freq]string{Daily: "day", Weekly: "week", Monthly: "month", Yearly: "year"}[r.Freq]
	if r.Interval == 1 {
		return "every " + unit
	}
	return fmt.Sprintf("every %d %ss", r.Interval, unit)
}

// Next returns the first occurrence strictly after the given time. Occurrence k
// is the anchor plus k intervals, so a monthly rule anchored on the 31st lands
// on the last day of shorter months and returns to the 31st afterwards.
func (r Rule) Next(anchor, after time.Time) time.Time {
	anchor, after = anchor.UTC(), after.UTC()
	if after.Before(anchor) {
		return anchor
	}
	switch r.Freq {
	case Daily, Weekly:
		step := time.Duration(r.Interval) * 24 * time.Hour
		if r.Freq == Weekly {
			step *= 7
		}
		return anchor.Add((after.Sub(anchor)/step + 1) * step)
	}

	months := r.Interval
	if r.Freq == Yearly {
		months *= 12
	}
	elapsed := (after.Year()-anchor.Year())*12 + int(after.Month()-anchor.Month())
	k := max(elapsed/months-1, 0)
	for {
		if occurrence := addMonths(anchor, k*months); occurrence.After(after) {
			return occurrence
		}
		k++
	}
}

func (r Rule) NextDate(anchor, after string) (string, error) {
	anchorTime, err := time.Parse(DateLayout, anchor)
	if err != nil {
		return "", err
	}
	afterTime, err := time.Parse(DateLayout, after)
	if err != nil {
		return "", err
	}
	return r.Next(anchorTime, afterTime).Format(DateLayout), nil
}

func addMonths(anchor time.Time, months int) time.Time {
	total := int(anchor.Month()) - 1 + months
	year := anchor.Year() + total/12
	month := time.Month(total%12 + 1)
	day := min(anchor.Day(), daysIn(year, month))
	return time.Date(year, month, day, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), time.UTC)
}

func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
