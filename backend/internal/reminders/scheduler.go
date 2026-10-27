package reminders

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alialjaffer/fai/backend/internal/mailer"
	"github.com/alialjaffer/fai/backend/internal/ntfy"
	"github.com/alialjaffer/fai/backend/internal/store"
)

type Scheduler struct {
	Store store.Store
	Ntfy  *ntfy.Client
	Mail  *mailer.Mailer
}

// Run ticks every minute checking for due, unsent reminders and delivering them
// via whichever channels are configured. No-ops when both are unconfigured.
// Blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	if s.Ntfy == nil && s.Mail == nil {
		return
	}

	due, err := s.Store.ListDueUnsentReminders(ctx)
	if err != nil {
		log.Printf("reminders: list due: %v", err)
	}
	for _, r := range due {
		if !s.notify(ctx, "reminders", r.ID, "Reminder", r.Title) {
			continue
		}
		if err := s.Store.MarkReminderSent(ctx, r.ID); err != nil {
			log.Printf("reminders: mark sent %d: %v", r.ID, err)
		}
	}

	pending, err := s.Store.ListDueUnpromptedMeetingOutcomes(ctx)
	if err != nil {
		log.Printf("meeting outcomes: list due: %v", err)
		return
	}
	for _, m := range pending {
		title := m.Title
		if title == "" {
			title = "Untitled meeting"
		}
		if !s.notify(ctx, "meeting outcomes", m.ID, "Meeting ended", fmt.Sprintf("'%s' just ended — log the outcome?", title)) {
			continue
		}
		if err := s.Store.MarkMeetingOutcomePromptSent(ctx, m.ID); err != nil {
			log.Printf("meeting outcomes: mark prompt sent %d: %v", m.ID, err)
		}
	}
}

// notify attempts every configured channel and reports whether at least one
// delivered. Marking the item sent after a partial failure avoids re-sending
// the channel that already succeeded on the next tick.
func (s *Scheduler) notify(ctx context.Context, kind string, id int64, subject, message string) bool {
	delivered := false
	if s.Ntfy != nil {
		if err := s.Ntfy.Publish(subject, message); err != nil {
			log.Printf("%s: ntfy %d: %v", kind, id, err)
		} else {
			delivered = true
		}
	}
	if s.Mail != nil {
		if err := s.Mail.Send(ctx, "fai: "+subject, message); err != nil {
			log.Printf("%s: email %d: %v", kind, id, err)
		} else {
			delivered = true
		}
	}
	return delivered
}
