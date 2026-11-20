package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alialjaffer/fai/backend/internal/recurrence"
	"github.com/alialjaffer/fai/backend/internal/store"
)

type repeatChange struct {
	set  bool
	rule *recurrence.Rule
}

func decodeRepeat(raw json.RawMessage) (repeatChange, error) {
	if raw == nil {
		return repeatChange{}, nil
	}
	if string(raw) == "null" {
		return repeatChange{set: true}, nil
	}
	var rule recurrence.Rule
	if err := json.Unmarshal(raw, &rule); err != nil {
		return repeatChange{}, fmt.Errorf("repeat must be an object with freq and interval, or null")
	}
	if err := rule.Validate(); err != nil {
		return repeatChange{}, err
	}
	return repeatChange{set: true, rule: &rule}, nil
}

const aiRepeatPrompt = `Add "repeat": {"freq": "daily"|"weekly"|"monthly"|"yearly", "interval": number} ` +
	"only when the message clearly asks for a repeat (every, daily, each week, every 2 weeks), with interval 1 or more. " +
	"Leave repeat out when the message does not ask for one, and also when the pattern does not fit one freq and interval " +
	"(every weekday, every Monday and Thursday). "

func modelRepeat(raw json.RawMessage) repeatChange {
	change, err := decodeRepeat(raw)
	if err != nil {
		return repeatChange{}
	}
	return change
}

func reminderRepeat(change repeatChange, remindAt time.Time) *store.ReminderRepeat {
	if change.rule == nil {
		return nil
	}
	return &store.ReminderRepeat{Rule: *change.rule, Anchor: remindAt}
}

func reminderRepeatForUpdate(current store.Reminder, remindAt time.Time, change repeatChange, reanchor bool) *store.ReminderRepeat {
	if change.set {
		return reminderRepeat(change, remindAt)
	}
	state := current.RepeatState()
	if state != nil && reanchor && !remindAt.Equal(current.RemindAt) {
		state.Anchor = remindAt
	}
	return state
}

func nextReminderTime(state store.ReminderRepeat, remindAt, now time.Time) time.Time {
	return state.Rule.Next(state.Anchor, laterOf(remindAt, now))
}

func laterOf(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func (s *Server) completeReminderItem(ctx context.Context, reminder store.Reminder, now time.Time) (store.Reminder, error) {
	state := reminder.RepeatState()
	if state == nil {
		return s.Store.SetReminderCompleted(ctx, reminder.ID, true)
	}
	return s.Store.AdvanceReminder(ctx, reminder.ID, nextReminderTime(*state, reminder.RemindAt, now))
}

func (s *Server) completeReminderByID(ctx context.Context, id int64) (store.Reminder, error) {
	reminder, err := s.Store.GetReminder(ctx, id)
	if err != nil {
		return store.Reminder{}, err
	}
	return s.completeReminderItem(ctx, reminder, time.Now())
}

var errRepeatNeedsDueDate = errors.New("a repeating task needs a due date")

func hasDate(due *string) bool {
	return due != nil && strings.TrimSpace(*due) != ""
}

func taskRepeatNeedsCurrent(patch store.TaskPatch, change repeatChange) bool {
	return change.rule != nil || patch.DueDateSet
}

func applyTaskRepeat(current store.Task, patch *store.TaskPatch, change repeatChange) error {
	if change.set && change.rule == nil {
		patch.RepeatSet = true
		return nil
	}
	due := current.DueDate
	if patch.DueDateSet {
		due = patch.DueDate
	}
	if change.rule != nil {
		if current.Kind == "arc" {
			return errors.New("an arc cannot repeat")
		}
		if !hasDate(due) {
			return errRepeatNeedsDueDate
		}
		if _, err := time.Parse(recurrence.DateLayout, *due); err != nil {
			return fmt.Errorf("invalid due date %q", *due)
		}
		patch.RepeatSet = true
		patch.Repeat = &store.TaskRepeat{Rule: *change.rule, Anchor: *due}
		return nil
	}
	if patch.DueDateSet && current.Repeat != nil {
		patch.RepeatSet = true
		if hasDate(due) {
			patch.Repeat = &store.TaskRepeat{Rule: *current.Repeat, Anchor: *due}
		}
	}
	return nil
}

func taskCreateRepeat(change repeatChange, kind string, due *string) (*store.TaskRepeat, error) {
	if change.rule == nil {
		return nil, nil
	}
	if kind == "arc" {
		return nil, errors.New("an arc cannot repeat")
	}
	if !hasDate(due) {
		return nil, errRepeatNeedsDueDate
	}
	if _, err := time.Parse(recurrence.DateLayout, *due); err != nil {
		return nil, fmt.Errorf("invalid due date %q", *due)
	}
	return &store.TaskRepeat{Rule: *change.rule, Anchor: *due}, nil
}

func importedTaskRepeat(item workspaceTaskExport) (*store.TaskRepeat, error) {
	if item.Repeat == nil {
		return nil, nil
	}
	if err := item.Repeat.Validate(); err != nil {
		return nil, err
	}
	repeat, err := taskCreateRepeat(repeatChange{set: true, rule: item.Repeat}, item.Kind, item.DueDate)
	if err != nil {
		return nil, err
	}
	if hasDate(item.RepeatAnchor) {
		if _, err := time.Parse(recurrence.DateLayout, *item.RepeatAnchor); err != nil {
			return nil, fmt.Errorf("invalid repeat anchor %q", *item.RepeatAnchor)
		}
		repeat.Anchor = *item.RepeatAnchor
	}
	return repeat, nil
}
