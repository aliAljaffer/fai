package reminders

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/ntfy"
)

type Scheduler struct {
	Q    *sqlc.Queries
	Ntfy *ntfy.Client
}

// Run ticks every minute checking for due, unsent reminders and pushing them via ntfy.
// No-ops gracefully if ntfy is unconfigured (Ntfy == nil). Blocks until ctx is cancelled.
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
	if s.Ntfy == nil {
		return
	}

	due, err := s.Q.ListDueUnsentReminders(ctx)
	if err != nil {
		log.Printf("reminders: list due: %v", err)
	}
	for _, r := range due {
		if err := s.Ntfy.Publish("Reminder", r.Title); err != nil {
			log.Printf("reminders: publish %d: %v", r.ID, err)
			continue
		}
		if err := s.Q.MarkReminderSent(ctx, r.ID); err != nil {
			log.Printf("reminders: mark sent %d: %v", r.ID, err)
		}
	}

	pending, err := s.Q.ListDueUnpromptedMeetingOutcomes(ctx)
	if err != nil {
		log.Printf("meeting outcomes: list due: %v", err)
		return
	}
	for _, m := range pending {
		title := m.Title
		if title == "" {
			title = "Untitled meeting"
		}
		if err := s.Ntfy.Publish("Meeting ended", fmt.Sprintf("'%s' just ended — log the outcome?", title)); err != nil {
			log.Printf("meeting outcomes: publish %d: %v", m.ID, err)
			continue
		}
		if err := s.Q.MarkMeetingOutcomePromptSent(ctx, m.ID); err != nil {
			log.Printf("meeting outcomes: mark prompt sent %d: %v", m.ID, err)
		}
	}
}
