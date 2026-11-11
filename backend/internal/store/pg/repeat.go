package pg

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/alialjaffer/fai/backend/internal/recurrence"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func repeatRule(freq pgtype.Text, interval pgtype.Int4) *recurrence.Rule {
	if !freq.Valid || !interval.Valid {
		return nil
	}
	return &recurrence.Rule{Freq: recurrence.Freq(freq.String), Interval: int(interval.Int32)}
}

func repeatColumns(rule *recurrence.Rule) (pgtype.Text, pgtype.Int4) {
	if rule == nil {
		return pgtype.Text{}, pgtype.Int4{}
	}
	return pgtype.Text{String: string(rule.Freq), Valid: true}, pgtype.Int4{Int32: int32(rule.Interval), Valid: true}
}

func taskRepeatColumns(repeat *store.TaskRepeat) (pgtype.Text, pgtype.Int4, pgtype.Date, error) {
	if repeat == nil {
		return pgtype.Text{}, pgtype.Int4{}, pgtype.Date{}, nil
	}
	freq, interval := repeatColumns(&repeat.Rule)
	anchor, err := ptrToDate(&repeat.Anchor)
	return freq, interval, anchor, err
}

func reminderRepeatColumns(repeat *store.ReminderRepeat) (pgtype.Text, pgtype.Int4, pgtype.Timestamptz) {
	if repeat == nil {
		return pgtype.Text{}, pgtype.Int4{}, pgtype.Timestamptz{}
	}
	freq, interval := repeatColumns(&repeat.Rule)
	return freq, interval, ptrToTs(&repeat.Anchor)
}
