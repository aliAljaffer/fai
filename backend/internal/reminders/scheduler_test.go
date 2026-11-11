package reminders

import (
	"context"
	"testing"
	"time"

	"github.com/alialjaffer/fai/backend/internal/recurrence"
	"github.com/alialjaffer/fai/backend/internal/store"
)

type fakeSettler struct {
	sent     []int64
	advanced map[int64]time.Time
}

func (f *fakeSettler) MarkReminderSent(_ context.Context, id int64) error {
	f.sent = append(f.sent, id)
	return nil
}

func (f *fakeSettler) AdvanceReminder(_ context.Context, id int64, remindAt time.Time) (store.Reminder, error) {
	if f.advanced == nil {
		f.advanced = map[int64]time.Time{}
	}
	f.advanced[id] = remindAt
	return store.Reminder{ID: id, RemindAt: remindAt}, nil
}

func TestSettleDeliveredMarksPlainRemindersSent(t *testing.T) {
	fake := &fakeSettler{}
	reminder := store.Reminder{ID: 1, RemindAt: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	if err := settleDelivered(t.Context(), fake, reminder, time.Date(2026, 3, 1, 9, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 1 || fake.sent[0] != 1 || len(fake.advanced) != 0 {
		t.Errorf("sent = %v, advanced = %v", fake.sent, fake.advanced)
	}
}

func TestSettleDeliveredAdvancesRepeatingReminders(t *testing.T) {
	anchor := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	reminder := store.Reminder{
		ID: 2, RemindAt: anchor,
		Repeat:       &recurrence.Rule{Freq: recurrence.Daily, Interval: 1},
		RepeatAnchor: &anchor,
	}
	fake := &fakeSettler{}
	now := time.Date(2026, 3, 4, 9, 2, 0, 0, time.UTC)
	if err := settleDelivered(t.Context(), fake, reminder, now); err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 3, 5, 9, 0, 0, 0, time.UTC)
	if got := fake.advanced[2]; !got.Equal(want) {
		t.Errorf("advanced to %s, want %s", got, want)
	}
	if len(fake.sent) != 0 {
		t.Errorf("a repeating reminder was marked sent: %v", fake.sent)
	}
}
