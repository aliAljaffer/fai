package lite

import (
	"database/sql"
	"time"
)

// timeLayout matches the strftime('%Y-%m-%dT%H:%M:%fZ', 'now') format used by every
// timestamp column/default in migrations_sqlite — RFC3339 with a '.sss' fractional
// seconds and 'Z' always written as UTC, so parsing with time.RFC3339Nano (which
// tolerates variable fractional-second precision) round-trips cleanly.
const timeLayout = time.RFC3339Nano

func parseTime(s string) (time.Time, error) {
	return time.Parse(timeLayout, s)
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

func nullTimeToPtr(s sql.NullString) *time.Time {
	if !s.Valid {
		return nil
	}
	t, err := parseTime(s.String)
	if err != nil {
		return nil
	}
	return &t
}

func ptrTimeToNull(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(*t), Valid: true}
}

func nullStringToPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func ptrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullInt64ToPtr(i sql.NullInt64) *int64 {
	if !i.Valid {
		return nil
	}
	return &i.Int64
}

func nowNullString() sql.NullString {
	t := time.Now()
	return sql.NullString{String: formatTime(t), Valid: true}
}

func ptrToNullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}
