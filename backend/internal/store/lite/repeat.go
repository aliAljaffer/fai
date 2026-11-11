package lite

import (
	"database/sql"

	"github.com/alialjaffer/fai/backend/internal/recurrence"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func repeatRule(freq sql.NullString, interval sql.NullInt64) *recurrence.Rule {
	if !freq.Valid || !interval.Valid {
		return nil
	}
	return &recurrence.Rule{Freq: recurrence.Freq(freq.String), Interval: int(interval.Int64)}
}

func repeatColumns(rule *recurrence.Rule) (sql.NullString, sql.NullInt64) {
	if rule == nil {
		return sql.NullString{}, sql.NullInt64{}
	}
	return sql.NullString{String: string(rule.Freq), Valid: true}, sql.NullInt64{Int64: int64(rule.Interval), Valid: true}
}

func taskRepeatColumns(repeat *store.TaskRepeat) (sql.NullString, sql.NullInt64, sql.NullString) {
	if repeat == nil {
		return sql.NullString{}, sql.NullInt64{}, sql.NullString{}
	}
	freq, interval := repeatColumns(&repeat.Rule)
	return freq, interval, sql.NullString{String: repeat.Anchor, Valid: true}
}
