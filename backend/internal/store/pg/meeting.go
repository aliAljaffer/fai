package pg

import (
	"context"
	"time"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toOutcome(m sqlc.MeetingOutcome) store.MeetingOutcome {
	return store.MeetingOutcome{
		ID: m.ID, OccurredOn: dateRequired(m.OccurredOn), Title: m.Title, NotesMd: m.NotesMd,
		EndsAt: tsToPtr(m.EndsAt), WorkspaceID: m.WorkspaceID, OutcomePromptSent: m.OutcomePromptSent,
		CalendarSourceID: int8ToPtr(m.CalendarSourceID), EventUID: m.EventUid, EventTitle: m.EventTitle,
		EventURL: m.EventUrl, CreatedAt: m.CreatedAt,
	}
}

func (s *Store) ListMeetingOutcomes(ctx context.Context, occurredOn *string, workspaceID *int64) ([]store.MeetingOutcome, error) {
	d, err := ptrToDate(occurredOn)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListMeetingOutcomes(ctx, sqlc.ListMeetingOutcomesParams{OccurredOn: d, WorkspaceID: ptrToInt8(workspaceID)})
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

func (s *Store) GetMeetingOutcomeByEvent(ctx context.Context, sourceID int64, eventUID string) (store.MeetingOutcome, error) {
	m, err := s.q.GetMeetingOutcomeByEvent(ctx, sqlc.GetMeetingOutcomeByEventParams{
		CalendarSourceID: ptrToInt8(&sourceID), EventUid: eventUID,
	})
	return toOutcome(m), err
}

func (s *Store) CreateMeetingOutcome(ctx context.Context, occurredOn, title, notesMd string, endsAt *time.Time, workspaceID int64, event store.MeetingEvent) (store.MeetingOutcome, error) {
	d, err := dateFromRequired(occurredOn)
	if err != nil {
		return store.MeetingOutcome{}, err
	}
	var sourceID *int64
	if event.UID != "" && event.SourceID != 0 {
		sourceID = &event.SourceID
	}
	m, err := s.q.CreateMeetingOutcome(ctx, sqlc.CreateMeetingOutcomeParams{
		OccurredOn: d, Title: title, NotesMd: notesMd, EndsAt: ptrToTs(endsAt), WorkspaceID: workspaceID,
		CalendarSourceID: ptrToInt8(sourceID), EventUid: event.UID, EventTitle: event.Title, EventUrl: event.URL,
	})
	return toOutcome(m), err
}

func (s *Store) UpdateMeetingOutcome(ctx context.Context, id int64, title, notesMd string, endsAt *time.Time) (store.MeetingOutcome, error) {
	m, err := s.q.UpdateMeetingOutcome(ctx, sqlc.UpdateMeetingOutcomeParams{
		ID: id, Title: title, NotesMd: notesMd, EndsAt: ptrToTs(endsAt),
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
