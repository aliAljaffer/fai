package tasks

import (
	"time"

	"github.com/alialjaffer/fai/backend/internal/recurrence"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func nextTaskDue(state store.TaskRepeat, due string, now time.Time) (string, error) {
	after := due
	if yesterday := now.UTC().AddDate(0, 0, -1).Format(recurrence.DateLayout); yesterday > after {
		after = yesterday
	}
	return state.Rule.NextDate(state.Anchor, after)
}
