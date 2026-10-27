package lite

import (
	"context"
	"time"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toOutcome(m sqlitedb.MeetingOutcome) store.MeetingOutcome {
	createdAt, _ := parseTime(m.CreatedAt)
	return store.MeetingOutcome{
		ID: m.ID, OccurredOn: m.OccurredOn, Title: m.Title, NotesMd: m.NotesMd,
		EndsAt: nullTimeToPtr(m.EndsAt), OutcomePromptSent: m.OutcomePromptSent, CreatedAt: createdAt,
	}
}

func (s *Store) ListMeetingOutcomes(ctx context.Context, occurredOn *string) ([]store.MeetingOutcome, error) {
	rows, err := s.q.ListMeetingOutcomes(ctx, ptrToNullString(occurredOn))
	if err != nil {
		return nil, err
	}
	out := make([]store.MeetingOutcome, len(rows))
	for i, m := range rows {
		out[i] = toOutcome(m)
	}
	return out, nil
}

func (s *Store) ListPendingMeetingOutcomes(ctx context.Context) ([]store.MeetingOutcome, error) {
	rows, err := s.q.ListPendingMeetingOutcomes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.MeetingOutcome, len(rows))
	for i, m := range rows {
		out[i] = toOutcome(m)
	}
	return out, nil
}

func (s *Store) GetMeetingOutcome(ctx context.Context, id int64) (store.MeetingOutcome, error) {
	m, err := s.q.GetMeetingOutcome(ctx, id)
	return toOutcome(m), err
}

func (s *Store) CreateMeetingOutcome(ctx context.Context, occurredOn, title, notesMd string, endsAt *time.Time) (store.MeetingOutcome, error) {
	m, err := s.q.CreateMeetingOutcome(ctx, sqlitedb.CreateMeetingOutcomeParams{
		OccurredOn: occurredOn, Title: title, NotesMd: notesMd, EndsAt: ptrTimeToNull(endsAt),
	})
	return toOutcome(m), err
}

func (s *Store) UpdateMeetingOutcome(ctx context.Context, id int64, title, notesMd string, endsAt *time.Time) (store.MeetingOutcome, error) {
	m, err := s.q.UpdateMeetingOutcome(ctx, sqlitedb.UpdateMeetingOutcomeParams{
		ID: id, Title: title, NotesMd: notesMd, EndsAt: ptrTimeToNull(endsAt),
	})
	return toOutcome(m), err
}

func (s *Store) DeleteMeetingOutcome(ctx context.Context, id int64) error {
	return s.q.DeleteMeetingOutcome(ctx, id)
}

func (s *Store) ListDueUnpromptedMeetingOutcomes(ctx context.Context) ([]store.MeetingOutcome, error) {
	rows, err := s.q.ListDueUnpromptedMeetingOutcomes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.MeetingOutcome, len(rows))
	for i, m := range rows {
		out[i] = toOutcome(m)
	}
	return out, nil
}

func (s *Store) MarkMeetingOutcomePromptSent(ctx context.Context, id int64) error {
	return s.q.MarkMeetingOutcomePromptSent(ctx, id)
}
