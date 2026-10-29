package pg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) DumpBackup(ctx context.Context) (store.BackupData, error) {
	var out store.BackupData

	rows, err := s.pool.Query(ctx, `SELECT id, name, tag, ref_seq, created_at FROM workspaces ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupWorkspace
		if err := rows.Scan(&item.ID, &item.Name, &item.Tag, &item.RefSeq, &item.CreatedAt); err != nil {
			rows.Close()
			return out, err
		}
		out.Workspaces = append(out.Workspaces, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `SELECT workspace_id, tag, created_at FROM workspace_tag_aliases ORDER BY workspace_id, tag`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupWorkspaceTagAlias
		if err := rows.Scan(&item.WorkspaceID, &item.Tag, &item.CreatedAt); err != nil {
			rows.Close()
			return out, err
		}
		out.WorkspaceTagAliases = append(out.WorkspaceTagAliases, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `SELECT id, name, workspace_id, created_at FROM boards ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupBoard
		if err := rows.Scan(&item.ID, &item.Name, &item.WorkspaceID, &item.CreatedAt); err != nil {
			rows.Close()
			return out, err
		}
		out.Boards = append(out.Boards, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `SELECT id, board_id, name, position, is_done, created_at FROM board_columns ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupColumn
		if err := rows.Scan(&item.ID, &item.BoardID, &item.Name, &item.Position, &item.IsDone, &item.CreatedAt); err != nil {
			rows.Close()
			return out, err
		}
		out.Columns = append(out.Columns, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `SELECT id, name, color FROM tags ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupTag
		if err := rows.Scan(&item.ID, &item.Name, &item.Color); err != nil {
			rows.Close()
			return out, err
		}
		out.Tags = append(out.Tags, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT id, board_id, column_id, workspace_id, ref_num, title, description_md, priority, due_date,
		       position, parent_task_id, kind, meeting_outcome_id, completed_at, archived, created_at, updated_at
		FROM tasks ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupTask
		var dueDate pgtype.Date
		var completedAt pgtype.Timestamptz
		if err := rows.Scan(
			&item.ID, &item.BoardID, &item.ColumnID, &item.WorkspaceID, &item.RefNum, &item.Title,
			&item.DescriptionMd, &item.Priority, &dueDate, &item.Position, &item.ParentTaskID, &item.Kind,
			&item.MeetingOutcomeID, &completedAt, &item.Archived, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			rows.Close()
			return out, err
		}
		item.DueDate = dateToPtr(dueDate)
		item.CompletedAt = tsToPtr(completedAt)
		out.Tasks = append(out.Tasks, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `SELECT task_id, tag_id FROM task_tags ORDER BY task_id, tag_id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupTaskTag
		if err := rows.Scan(&item.TaskID, &item.TagID); err != nil {
			rows.Close()
			return out, err
		}
		out.TaskTags = append(out.TaskTags, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `SELECT id, title, body_md, task_id, workspace_id, created_at, updated_at FROM notes ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupNote
		if err := rows.Scan(&item.ID, &item.Title, &item.BodyMd, &item.TaskID, &item.WorkspaceID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			rows.Close()
			return out, err
		}
		out.Notes = append(out.Notes, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `SELECT note_id, tag_id FROM note_tags ORDER BY note_id, tag_id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupNoteTag
		if err := rows.Scan(&item.NoteID, &item.TagID); err != nil {
			rows.Close()
			return out, err
		}
		out.NoteTags = append(out.NoteTags, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `SELECT id, title, remind_at, task_id, workspace_id, sent, created_at, completed_at FROM reminders ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupReminder
		var completedAt pgtype.Timestamptz
		if err := rows.Scan(&item.ID, &item.Title, &item.RemindAt, &item.TaskID, &item.WorkspaceID, &item.Sent, &item.CreatedAt, &completedAt); err != nil {
			rows.Close()
			return out, err
		}
		item.CompletedAt = tsToPtr(completedAt)
		out.Reminders = append(out.Reminders, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `SELECT id, task_id, started_at, ended_at FROM task_time_entries ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupTimeEntry
		var endedAt pgtype.Timestamptz
		if err := rows.Scan(&item.ID, &item.TaskID, &item.StartedAt, &endedAt); err != nil {
			rows.Close()
			return out, err
		}
		item.EndedAt = tsToPtr(endedAt)
		out.TimeEntries = append(out.TimeEntries, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at
		FROM jira_links ORDER BY task_id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupJiraLink
		var lastImported, lastExported pgtype.Timestamptz
		var rawStatus pgtype.Text
		if err := rows.Scan(&item.TaskID, &item.JiraKey, &item.JiraURL, &item.JiraProjectKey, &item.LinkOrigin,
			&lastImported, &lastExported, &rawStatus, &item.CreatedAt); err != nil {
			rows.Close()
			return out, err
		}
		item.LastImportedAt = tsToPtr(lastImported)
		item.LastExportedAt = tsToPtr(lastExported)
		item.RawStatus = textToPtr(rawStatus)
		out.JiraLinks = append(out.JiraLinks, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT id, task_id, note_id, filename, content_type, size_bytes, storage_key, created_at
		FROM attachments ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupAttachment
		if err := rows.Scan(&item.ID, &item.TaskID, &item.NoteID, &item.Filename, &item.ContentType,
			&item.SizeBytes, &item.StorageKey, &item.CreatedAt); err != nil {
			rows.Close()
			return out, err
		}
		out.Attachments = append(out.Attachments, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `SELECT id, name, url, color, workspace_id, created_at FROM calendar_sources ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupCalendarSource
		if err := rows.Scan(&item.ID, &item.Name, &item.URL, &item.Color, &item.WorkspaceID, &item.CreatedAt); err != nil {
			rows.Close()
			return out, err
		}
		out.CalendarSources = append(out.CalendarSources, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT id, occurred_on, title, notes_md, ends_at, workspace_id, outcome_prompt_sent, created_at
		FROM meeting_outcomes ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupMeetingOutcome
		var occurredOn pgtype.Date
		var endsAt pgtype.Timestamptz
		if err := rows.Scan(&item.ID, &occurredOn, &item.Title, &item.NotesMd, &endsAt,
			&item.WorkspaceID, &item.OutcomePromptSent, &item.CreatedAt); err != nil {
			rows.Close()
			return out, err
		}
		item.OccurredOn = dateRequired(occurredOn)
		item.EndsAt = tsToPtr(endsAt)
		out.MeetingOutcomes = append(out.MeetingOutcomes, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `SELECT id, task_id, author, body_md, external_id, created_at FROM task_comments ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupTaskComment
		var externalID pgtype.Text
		if err := rows.Scan(&item.ID, &item.TaskID, &item.Author, &item.BodyMd, &externalID, &item.CreatedAt); err != nil {
			rows.Close()
			return out, err
		}
		item.ExternalID = textToPtr(externalID)
		out.TaskComments = append(out.TaskComments, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	var ai store.BackupAISettings
	err = s.pool.QueryRow(ctx, `SELECT provider, base_url, model, updated_at FROM ai_settings WHERE id = 1`).
		Scan(&ai.Provider, &ai.BaseURL, &ai.Model, &ai.UpdatedAt)
	if err == nil {
		out.AISettings = &ai
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}

	appearance, err := s.GetAppearance(ctx)
	if err != nil {
		return out, err
	}
	out.Appearance = &appearance

	return out, nil
}

func (s *Store) RestoreBackup(ctx context.Context, data store.BackupData) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	deletes := []string{
		`DELETE FROM calendar_feed_cache`,
		`DELETE FROM task_comments`,
		`DELETE FROM attachments`,
		`DELETE FROM note_tags`,
		`DELETE FROM task_tags`,
		`DELETE FROM task_time_entries`,
		`DELETE FROM jira_links`,
		`DELETE FROM notes`,
		`DELETE FROM reminders`,
		`DELETE FROM tasks`,
		`DELETE FROM meeting_outcomes`,
		`DELETE FROM board_columns`,
		`DELETE FROM boards`,
		`DELETE FROM calendar_sources`,
		`DELETE FROM tags`,
		`DELETE FROM workspace_themes`,
		`DELETE FROM appearance_settings`,
		`DELETE FROM ai_settings`,
		`DELETE FROM workspace_tag_aliases`,
		`DELETE FROM workspaces`,
	}
	for _, query := range deletes {
		if _, err := tx.Exec(ctx, query); err != nil {
			return err
		}
	}

	for _, item := range data.Workspaces {
		if _, err := tx.Exec(ctx,
			`INSERT INTO workspaces (id, name, tag, ref_seq, created_at) OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4, $5)`,
			item.ID, item.Name, item.Tag, item.RefSeq, item.CreatedAt); err != nil {
			return err
		}
	}
	for _, item := range data.WorkspaceTagAliases {
		if _, err := tx.Exec(ctx,
			`INSERT INTO workspace_tag_aliases (workspace_id, tag, created_at) VALUES ($1, $2, $3)`,
			item.WorkspaceID, item.Tag, item.CreatedAt); err != nil {
			return err
		}
	}
	for _, item := range data.Boards {
		if _, err := tx.Exec(ctx,
			`INSERT INTO boards (id, name, workspace_id, created_at) OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4)`,
			item.ID, item.Name, item.WorkspaceID, item.CreatedAt); err != nil {
			return err
		}
	}
	for _, item := range data.Columns {
		if _, err := tx.Exec(ctx,
			`INSERT INTO board_columns (id, board_id, name, position, is_done, created_at) OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4, $5, $6)`,
			item.ID, item.BoardID, item.Name, item.Position, item.IsDone, item.CreatedAt); err != nil {
			return err
		}
	}
	for _, item := range data.Tags {
		if _, err := tx.Exec(ctx,
			`INSERT INTO tags (id, name, color) OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3)`,
			item.ID, item.Name, item.Color); err != nil {
			return err
		}
	}
	for _, item := range data.MeetingOutcomes {
		occurredOn, err := dateFromRequired(item.OccurredOn)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO meeting_outcomes (id, occurred_on, title, notes_md, ends_at, workspace_id, outcome_prompt_sent, created_at)
			 OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			item.ID, occurredOn, item.Title, item.NotesMd, ptrToTs(item.EndsAt),
			item.WorkspaceID, item.OutcomePromptSent, item.CreatedAt); err != nil {
			return err
		}
	}
	for _, item := range data.Tasks {
		dueDate, err := ptrToDate(item.DueDate)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO tasks (id, board_id, column_id, workspace_id, ref_num, title, description_md, priority, due_date,
				position, parent_task_id, kind, meeting_outcome_id, completed_at, archived, created_at, updated_at)
			OVERRIDING SYSTEM VALUE
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULL, $11, $12, $13, $14, $15, $16)`,
			item.ID, item.BoardID, item.ColumnID, item.WorkspaceID, item.RefNum, item.Title, item.DescriptionMd,
			item.Priority, dueDate, item.Position, item.Kind, ptrToInt8(item.MeetingOutcomeID),
			ptrToTs(item.CompletedAt), item.Archived, item.CreatedAt, item.UpdatedAt); err != nil {
			return err
		}
	}
	for _, item := range data.Tasks {
		if item.ParentTaskID == nil {
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE tasks SET parent_task_id = $1 WHERE id = $2`, *item.ParentTaskID, item.ID); err != nil {
			return err
		}
	}
	for _, item := range data.TaskTags {
		if _, err := tx.Exec(ctx, `INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2)`, item.TaskID, item.TagID); err != nil {
			return err
		}
	}
	for _, item := range data.Notes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO notes (id, title, body_md, task_id, workspace_id, created_at, updated_at) OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			item.ID, item.Title, item.BodyMd, ptrToInt8(item.TaskID), item.WorkspaceID, item.CreatedAt, item.UpdatedAt); err != nil {
			return err
		}
	}
	for _, item := range data.NoteTags {
		if _, err := tx.Exec(ctx, `INSERT INTO note_tags (note_id, tag_id) VALUES ($1, $2)`, item.NoteID, item.TagID); err != nil {
			return err
		}
	}
	for _, item := range data.Reminders {
		if _, err := tx.Exec(ctx,
			`INSERT INTO reminders (id, title, remind_at, task_id, workspace_id, sent, created_at, completed_at) OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			item.ID, item.Title, item.RemindAt, ptrToInt8(item.TaskID), item.WorkspaceID, item.Sent, item.CreatedAt, ptrToTs(item.CompletedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.TimeEntries {
		if _, err := tx.Exec(ctx,
			`INSERT INTO task_time_entries (id, task_id, started_at, ended_at) OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4)`,
			item.ID, item.TaskID, item.StartedAt, ptrToTs(item.EndedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.JiraLinks {
		if _, err := tx.Exec(ctx, `
			INSERT INTO jira_links (task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			item.TaskID, item.JiraKey, item.JiraURL, item.JiraProjectKey, item.LinkOrigin,
			ptrToTs(item.LastImportedAt), ptrToTs(item.LastExportedAt), ptrToText(item.RawStatus),
			item.CreatedAt); err != nil {
			return err
		}
	}
	for _, item := range data.Attachments {
		if _, err := tx.Exec(ctx, `
			INSERT INTO attachments (id, task_id, note_id, filename, content_type, size_bytes, storage_key, created_at)
			OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			item.ID, ptrToInt8(item.TaskID), ptrToInt8(item.NoteID), item.Filename, item.ContentType,
			item.SizeBytes, item.StorageKey, item.CreatedAt); err != nil {
			return err
		}
	}
	for _, item := range data.CalendarSources {
		if _, err := tx.Exec(ctx,
			`INSERT INTO calendar_sources (id, name, url, color, workspace_id, created_at) OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4, $5, $6)`,
			item.ID, item.Name, item.URL, item.Color, item.WorkspaceID, item.CreatedAt); err != nil {
			return err
		}
	}
	for _, item := range data.TaskComments {
		if _, err := tx.Exec(ctx,
			`INSERT INTO task_comments (id, task_id, author, body_md, external_id, created_at) OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4, $5, $6)`,
			item.ID, item.TaskID, item.Author, item.BodyMd, ptrToText(item.ExternalID), item.CreatedAt); err != nil {
			return err
		}
	}
	if data.AISettings != nil {
		if _, err := tx.Exec(ctx,
			`INSERT INTO ai_settings (id, provider, base_url, model, api_key, updated_at) VALUES (1, $1, $2, $3, '', $4)`,
			data.AISettings.Provider, data.AISettings.BaseURL, data.AISettings.Model, data.AISettings.UpdatedAt); err != nil {
			return err
		}
	}
	if data.Appearance != nil {
		payload, err := json.Marshal(data.Appearance.CustomPalettes)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO appearance_settings (id, global_palette_id, custom_palettes, updated_at) VALUES (1, $1, $2, $3)`,
			data.Appearance.GlobalPaletteID, string(payload), data.Appearance.UpdatedAt); err != nil {
			return err
		}
		for workspace, paletteID := range data.Appearance.WorkspacePalettes {
			workspaceID, err := strconv.ParseInt(workspace, 10, 64)
			if err != nil {
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO workspace_themes (workspace_id, palette_id) VALUES ($1, $2)`,
				workspaceID, paletteID); err != nil {
				return err
			}
		}
	}

	tables := []string{
		"workspaces", "boards", "board_columns", "tasks", "tags", "notes", "reminders",
		"task_time_entries", "attachments", "calendar_sources", "meeting_outcomes", "task_comments",
	}
	for _, table := range tables {
		query := fmt.Sprintf(`SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE((SELECT max(id) FROM %s), 0) + 1, false)`, table, table)
		if _, err := tx.Exec(ctx, query); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
