package pg

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/alialjaffer/fai/backend/internal/db/sqlc"
	"github.com/alialjaffer/fai/backend/internal/store"
)

func toJiraLink(j sqlc.JiraLink) store.JiraLink {
	return store.JiraLink{
		TaskID: j.TaskID, JiraKey: j.JiraKey, JiraURL: j.JiraUrl, JiraProjectKey: j.JiraProjectKey,
		LinkOrigin: j.LinkOrigin, LastImportedAt: tsToPtr(j.LastImportedAt), LastExportedAt: tsToPtr(j.LastExportedAt),
		RawStatus: textToPtr(j.RawStatus), CreatedAt: j.CreatedAt,
	}
}

func (s *Store) GetJiraLinkByKey(ctx context.Context, jiraKey string) (*store.JiraLink, error) {
	j, err := s.q.GetJiraLinkByKey(ctx, jiraKey)
	if errors.Is(err, pgx.ErrNoRows) {
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
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	l := toJiraLink(j)
	return &l, nil
}

func (s *Store) ListJiraLinksByWorkspace(ctx context.Context, workspaceID *int64) ([]store.JiraLink, error) {
	rows, err := s.q.ListJiraLinksByWorkspace(ctx, ptrToInt8(workspaceID))
	if err != nil {
		return nil, err
	}
	out := make([]store.JiraLink, len(rows))
	for i, j := range rows {
		out[i] = toJiraLink(j)
	}
	return out, nil
}

func (s *Store) GetJiraTaskRef(ctx context.Context, jiraKey string) (*store.TaskRef, error) {
	row, err := s.q.GetJiraTaskRef(ctx, jiraKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ref := store.TaskRef{Kind: "task", ID: row.ID, Title: row.Title, RefNum: row.RefNum, WorkspaceTag: row.Tag, WorkspaceName: row.Name}
	ref.Ref = ref.WorkspaceTag + "-" + strconv.FormatInt(ref.RefNum, 10)
	return &ref, nil
}

func (s *Store) CreateJiraLinkImported(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey, rawStatus string) (store.JiraLink, error) {
	j, err := s.q.CreateJiraLink(ctx, sqlc.CreateJiraLinkParams{
		TaskID: taskID, JiraKey: jiraKey, JiraUrl: jiraURL, JiraProjectKey: projectKey,
		LinkOrigin: "imported", LastImportedAt: pgtimestamptzNow(), RawStatus: ptrToText(&rawStatus),
	})
	return toJiraLink(j), err
}

func (s *Store) CreateJiraLinkManual(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey, rawStatus string) (store.JiraLink, error) {
	j, err := s.q.CreateJiraLinkManual(ctx, sqlc.CreateJiraLinkManualParams{
		TaskID: taskID, JiraKey: jiraKey, JiraUrl: jiraURL, JiraProjectKey: projectKey, RawStatus: ptrToText(&rawStatus),
	})
	return toJiraLink(j), err
}

func (s *Store) CreateJiraLinkExported(ctx context.Context, taskID int64, jiraKey, jiraURL, projectKey string) (store.JiraLink, error) {
	j, err := s.q.CreateJiraLinkExported(ctx, sqlc.CreateJiraLinkExportedParams{
		TaskID: taskID, JiraKey: jiraKey, JiraUrl: jiraURL, JiraProjectKey: projectKey, LastExportedAt: pgtimestamptzNow(),
	})
	return toJiraLink(j), err
}

func (s *Store) UpdateJiraLinkImported(ctx context.Context, taskID int64, rawStatus string) error {
	return s.q.UpdateJiraLinkImported(ctx, sqlc.UpdateJiraLinkImportedParams{TaskID: taskID, RawStatus: ptrToText(&rawStatus)})
}

func (s *Store) DeleteJiraLink(ctx context.Context, taskID int64) error {
	return s.q.DeleteJiraLink(ctx, taskID)
}
