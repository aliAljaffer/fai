package reminders

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alialjaffer/fai/backend/internal/mailer"
	"github.com/alialjaffer/fai/backend/internal/ntfy"
	"github.com/alialjaffer/fai/backend/internal/store"
	"github.com/alialjaffer/fai/backend/internal/telegram"
)

type Scheduler struct {
	Store    store.Store
	Ntfy     *ntfy.Client
	Mail     *mailer.Mailer
	Telegram *telegram.Notifier
	Channel  string
}

// Run ticks every minute checking for due, unsent reminders and delivering them
// via the selected channel, or every configured channel when none is selected.
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

type reminderSettler interface {
	MarkReminderSent(ctx context.Context, id int64) error
	AdvanceReminder(ctx context.Context, id int64, remindAt time.Time) (store.Reminder, error)
}

func settleDelivered(ctx context.Context, st reminderSettler, r store.Reminder, now time.Time) error {
	if state := r.RepeatState(); state != nil {
		_, err := st.AdvanceReminder(ctx, r.ID, state.Rule.Next(state.Anchor, now))
		return err
	}
	return st.MarkReminderSent(ctx, r.ID)
}

func (s *Scheduler) tick(ctx context.Context) {
	if s.Ntfy == nil && s.Mail == nil && s.Telegram == nil {
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
		if err := settleDelivered(ctx, s.Store, r, time.Now()); err != nil {
			log.Printf("reminders: settle %d: %v", r.ID, err)
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
		if !s.notify(ctx, "meeting outcomes", m.ID, "Meeting ended", fmt.Sprintf("'%s' just ended, log the outcome?", title)) {
			continue
		}
		if err := s.Store.MarkMeetingOutcomePromptSent(ctx, m.ID); err != nil {
			log.Printf("meeting outcomes: mark prompt sent %d: %v", m.ID, err)
		}
	}
}

// notify reports whether at least one selected channel delivered. Marking the
// item sent after a partial failure avoids re-sending a channel that already
// succeeded on the next tick.
func (s *Scheduler) notify(ctx context.Context, kind string, id int64, subject, message string) bool {
	var delivered atomic.Bool
	sendNtfy, sendMail, sendTelegram := true, true, true
	switch s.Channel {
	case "":
	case "ntfy":
		sendMail, sendTelegram = false, false
	case "email", "mail", "smtp":
		sendNtfy, sendTelegram = false, false
	case "telegram":
		sendNtfy, sendMail = false, false
	default:
		log.Printf("reminders: unknown channel %q, using every configured channel", s.Channel)
	}

	var wg sync.WaitGroup
	if sendNtfy && s.Ntfy != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Ntfy.Publish(subject, message); err != nil {
				log.Printf("%s: ntfy %d: %v", kind, id, err)
			} else {
				delivered.Store(true)
			}
		}()
	}
	if sendMail && s.Mail != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Mail.Send(ctx, "fai: "+subject, message); err != nil {
				log.Printf("%s: email %d: %v", kind, id, err)
			} else {
				delivered.Store(true)
			}
		}()
	}
	if sendTelegram && s.Telegram != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Telegram.Publish(subject, message); err != nil {
				log.Printf("%s: telegram %d: %v", kind, id, err)
			} else {
				delivered.Store(true)
			}
		}()
	}
	wg.Wait()
	return delivered.Load()
}
