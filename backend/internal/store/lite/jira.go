package lite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toJiraLink(j sqlitedb.JiraLink) store.JiraLink {
	createdAt, _ := parseTime(j.CreatedAt)
	return store.JiraLink{
		TaskID: j.TaskID, JiraKey: j.JiraKey, JiraURL: j.JiraUrl, JiraProjectKey: j.JiraProjectKey,
		LinkOrigin: j.LinkOrigin, LastImportedAt: nullTimeToPtr(j.LastImportedAt), LastExportedAt: nullTimeToPtr(j.LastExportedAt),
		RawStatus: nullStringToPtr(j.RawStatus), CreatedAt: createdAt,
	}
}

func (s *Store) GetJiraLinkByKey(ctx context.Context, jiraKey string) (*store.JiraLink, error) {
	j, err := s.q.GetJiraLinkByKey(ctx, jiraKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	l := toJiraLink(j)
	return &l, nil
}

func (s *Store) GetJiraLinkByTask(ctx context.Context, taskID int64) (*store.JiraLink, error) {
	j, err := s.q.GetJiraLinkByTask(ctx, taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	l := toJiraLink(j)
	return &l, nil
}

func (s *Store) CreateJiraLinkImported(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey, rawStatus string) (store.JiraLink, error) {
	now := nowNullString()
	j, err := s.q.CreateJiraLink(ctx, sqlitedb.CreateJiraLinkParams{
		TaskID: taskID, JiraKey: jiraKey, JiraUrl: jiraURL, JiraProjectKey: projectKey,
		LinkOrigin: "imported", LastImportedAt: now, RawStatus: ptrToNullString(&rawStatus),
	})
	return toJiraLink(j), err
}

func (s *Store) CreateJiraLinkManual(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey, rawStatus string) (store.JiraLink, error) {
	j, err := s.q.CreateJiraLinkManual(ctx, sqlitedb.CreateJiraLinkManualParams{
		TaskID: taskID, JiraKey: jiraKey, JiraUrl: jiraURL, JiraProjectKey: projectKey, RawStatus: ptrToNullString(&rawStatus),
	})
	return toJiraLink(j), err
}

func (s *Store) CreateJiraLinkExported(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey string) (store.JiraLink, error) {
	j, err := s.q.CreateJiraLinkExported(ctx, sqlitedb.CreateJiraLinkExportedParams{
		TaskID: taskID, JiraKey: jiraKey, JiraUrl: jiraURL, JiraProjectKey: projectKey, LastExportedAt: nowNullString(),
	})
	return toJiraLink(j), err
}

func (s *Store) UpdateJiraLinkImported(ctx context.Context, taskID int64, rawStatus string) error {
	return s.q.UpdateJiraLinkImported(ctx, sqlitedb.UpdateJiraLinkImportedParams{TaskID: taskID, RawStatus: ptrToNullString(&rawStatus)})
}

func (s *Store) DeleteJiraLink(ctx context.Context, taskID int64) error {
	return s.q.DeleteJiraLink(ctx, taskID)
}
