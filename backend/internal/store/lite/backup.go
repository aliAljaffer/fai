package lite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func (s *Store) DumpBackup(ctx context.Context) (store.BackupData, error) {
	var out store.BackupData

	rows, err := s.db.QueryContext(ctx, `SELECT id, name, tag, ref_seq, created_at FROM workspaces ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupWorkspace
		var createdAt string
		if err := rows.Scan(&item.ID, &item.Name, &item.Tag, &item.RefSeq, &createdAt); err != nil {
			rows.Close()
			return out, err
		}
		item.CreatedAt, _ = parseTime(createdAt)
		out.Workspaces = append(out.Workspaces, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT workspace_id, tag, created_at FROM workspace_tag_aliases ORDER BY workspace_id, tag`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupWorkspaceTagAlias
		var createdAt string
		if err := rows.Scan(&item.WorkspaceID, &item.Tag, &createdAt); err != nil {
			rows.Close()
			return out, err
		}
		item.CreatedAt, _ = parseTime(createdAt)
		out.WorkspaceTagAliases = append(out.WorkspaceTagAliases, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT id, name, workspace_id, created_at FROM boards ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupBoard
		var createdAt string
		if err := rows.Scan(&item.ID, &item.Name, &item.WorkspaceID, &createdAt); err != nil {
			rows.Close()
			return out, err
		}
		item.CreatedAt, _ = parseTime(createdAt)
		out.Boards = append(out.Boards, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT id, board_id, name, position, is_done, created_at FROM board_columns ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupColumn
		var createdAt string
		if err := rows.Scan(&item.ID, &item.BoardID, &item.Name, &item.Position, &item.IsDone, &createdAt); err != nil {
			rows.Close()
			return out, err
		}
		item.CreatedAt, _ = parseTime(createdAt)
		out.Columns = append(out.Columns, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT id, name, color FROM tags ORDER BY id`)
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

	rows, err = s.db.QueryContext(ctx, `
		SELECT id, board_id, column_id, workspace_id, ref_num, title, description_md, priority, due_date,
		       position, parent_task_id, kind, meeting_outcome_id, completed_at, archived, created_at, updated_at
		FROM tasks ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupTask
		var dueDate, completedAt sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(
			&item.ID, &item.BoardID, &item.ColumnID, &item.WorkspaceID, &item.RefNum, &item.Title,
			&item.DescriptionMd, &item.Priority, &dueDate, &item.Position, &item.ParentTaskID, &item.Kind,
			&item.MeetingOutcomeID, &completedAt, &item.Archived, &createdAt, &updatedAt,
		); err != nil {
			rows.Close()
			return out, err
		}
		item.DueDate = nullStringToPtr(dueDate)
		item.CompletedAt = nullTimeToPtr(completedAt)
		item.CreatedAt, _ = parseTime(createdAt)
		item.UpdatedAt, _ = parseTime(updatedAt)
		out.Tasks = append(out.Tasks, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT task_id, tag_id FROM task_tags ORDER BY task_id, tag_id`)
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

	rows, err = s.db.QueryContext(ctx, `SELECT id, title, body_md, task_id, workspace_id, created_at, updated_at FROM notes ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupNote
		var createdAt, updatedAt string
		if err := rows.Scan(&item.ID, &item.Title, &item.BodyMd, &item.TaskID, &item.WorkspaceID, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return out, err
		}
		item.CreatedAt, _ = parseTime(createdAt)
		item.UpdatedAt, _ = parseTime(updatedAt)
		out.Notes = append(out.Notes, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT note_id, tag_id FROM note_tags ORDER BY note_id, tag_id`)
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

	rows, err = s.db.QueryContext(ctx, `SELECT id, title, remind_at, task_id, workspace_id, sent, created_at FROM reminders ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupReminder
		var remindAt, createdAt string
		if err := rows.Scan(&item.ID, &item.Title, &remindAt, &item.TaskID, &item.WorkspaceID, &item.Sent, &createdAt); err != nil {
			rows.Close()
			return out, err
		}
		item.RemindAt, _ = parseTime(remindAt)
		item.CreatedAt, _ = parseTime(createdAt)
		out.Reminders = append(out.Reminders, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT id, task_id, started_at, ended_at FROM task_time_entries ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupTimeEntry
		var startedAt string
		var endedAt sql.NullString
		if err := rows.Scan(&item.ID, &item.TaskID, &startedAt, &endedAt); err != nil {
			rows.Close()
			return out, err
		}
		item.StartedAt, _ = parseTime(startedAt)
		item.EndedAt = nullTimeToPtr(endedAt)
		out.TimeEntries = append(out.TimeEntries, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `
		SELECT task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at
		FROM jira_links ORDER BY task_id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupJiraLink
		var lastImported, lastExported, rawStatus sql.NullString
		var createdAt string
		if err := rows.Scan(&item.TaskID, &item.JiraKey, &item.JiraURL, &item.JiraProjectKey, &item.LinkOrigin,
			&lastImported, &lastExported, &rawStatus, &createdAt); err != nil {
			rows.Close()
			return out, err
		}
		item.LastImportedAt = nullTimeToPtr(lastImported)
		item.LastExportedAt = nullTimeToPtr(lastExported)
		item.RawStatus = nullStringToPtr(rawStatus)
		item.CreatedAt, _ = parseTime(createdAt)
		out.JiraLinks = append(out.JiraLinks, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `
		SELECT id, task_id, note_id, filename, content_type, size_bytes, storage_key, created_at
		FROM attachments ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupAttachment
		var createdAt string
		if err := rows.Scan(&item.ID, &item.TaskID, &item.NoteID, &item.Filename, &item.ContentType,
			&item.SizeBytes, &item.StorageKey, &createdAt); err != nil {
			rows.Close()
			return out, err
		}
		item.CreatedAt, _ = parseTime(createdAt)
		out.Attachments = append(out.Attachments, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT id, name, url, color, workspace_id, created_at FROM calendar_sources ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupCalendarSource
		var createdAt string
		if err := rows.Scan(&item.ID, &item.Name, &item.URL, &item.Color, &item.WorkspaceID, &createdAt); err != nil {
			rows.Close()
			return out, err
		}
		item.CreatedAt, _ = parseTime(createdAt)
		out.CalendarSources = append(out.CalendarSources, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `
		SELECT id, occurred_on, title, notes_md, ends_at, workspace_id, outcome_prompt_sent, created_at
		FROM meeting_outcomes ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupMeetingOutcome
		var endsAt sql.NullString
		var createdAt string
		if err := rows.Scan(&item.ID, &item.OccurredOn, &item.Title, &item.NotesMd, &endsAt,
			&item.WorkspaceID, &item.OutcomePromptSent, &createdAt); err != nil {
			rows.Close()
			return out, err
		}
		item.EndsAt = nullTimeToPtr(endsAt)
		item.CreatedAt, _ = parseTime(createdAt)
		out.MeetingOutcomes = append(out.MeetingOutcomes, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT id, task_id, author, body_md, external_id, created_at FROM task_comments ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item store.BackupTaskComment
		var externalID sql.NullString
		var createdAt string
		if err := rows.Scan(&item.ID, &item.TaskID, &item.Author, &item.BodyMd, &externalID, &createdAt); err != nil {
			rows.Close()
			return out, err
		}
		item.ExternalID = nullStringToPtr(externalID)
		item.CreatedAt, _ = parseTime(createdAt)
		out.TaskComments = append(out.TaskComments, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	var ai store.BackupAISettings
	var aiUpdatedAt string
	err = s.db.QueryRowContext(ctx, `SELECT provider, base_url, model, updated_at FROM ai_settings WHERE id = 1`).
		Scan(&ai.Provider, &ai.BaseURL, &ai.Model, &aiUpdatedAt)
	if err == nil {
		ai.UpdatedAt, _ = parseTime(aiUpdatedAt)
		out.AISettings = &ai
	} else if err != sql.ErrNoRows {
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return err
		}
	}

	for _, item := range data.Workspaces {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO workspaces (id, name, tag, ref_seq, created_at) VALUES (?, ?, ?, ?, ?)`,
			item.ID, item.Name, item.Tag, item.RefSeq, formatTime(item.CreatedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.WorkspaceTagAliases {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO workspace_tag_aliases (workspace_id, tag, created_at) VALUES (?, ?, ?)`,
			item.WorkspaceID, item.Tag, formatTime(item.CreatedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.Boards {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO boards (id, name, workspace_id, created_at) VALUES (?, ?, ?, ?)`,
			item.ID, item.Name, item.WorkspaceID, formatTime(item.CreatedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.Columns {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO board_columns (id, board_id, name, position, is_done, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			item.ID, item.BoardID, item.Name, item.Position, item.IsDone, formatTime(item.CreatedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.Tags {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO tags (id, name, color) VALUES (?, ?, ?)`,
			item.ID, item.Name, item.Color); err != nil {
			return err
		}
	}
	for _, item := range data.MeetingOutcomes {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO meeting_outcomes (id, occurred_on, title, notes_md, ends_at, workspace_id, outcome_prompt_sent, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.OccurredOn, item.Title, item.NotesMd, ptrTimeToNull(item.EndsAt),
			item.WorkspaceID, item.OutcomePromptSent, formatTime(item.CreatedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.Tasks {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tasks (id, board_id, column_id, workspace_id, ref_num, title, description_md, priority, due_date,
				position, parent_task_id, kind, meeting_outcome_id, completed_at, archived, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.BoardID, item.ColumnID, item.WorkspaceID, item.RefNum, item.Title, item.DescriptionMd,
			item.Priority, ptrToNullString(item.DueDate), item.Position, item.Kind, ptrToNullInt64(item.MeetingOutcomeID),
			ptrTimeToNull(item.CompletedAt), item.Archived, formatTime(item.CreatedAt), formatTime(item.UpdatedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.Tasks {
		if item.ParentTaskID == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET parent_task_id = ? WHERE id = ?`, *item.ParentTaskID, item.ID); err != nil {
			return err
		}
	}
	for _, item := range data.TaskTags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_tags (task_id, tag_id) VALUES (?, ?)`, item.TaskID, item.TagID); err != nil {
			return err
		}
	}
	for _, item := range data.Notes {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO notes (id, title, body_md, task_id, workspace_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.Title, item.BodyMd, ptrToNullInt64(item.TaskID), item.WorkspaceID,
			formatTime(item.CreatedAt), formatTime(item.UpdatedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.NoteTags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO note_tags (note_id, tag_id) VALUES (?, ?)`, item.NoteID, item.TagID); err != nil {
			return err
		}
	}
	for _, item := range data.Reminders {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO reminders (id, title, remind_at, task_id, workspace_id, sent, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.Title, formatTime(item.RemindAt), ptrToNullInt64(item.TaskID), item.WorkspaceID,
			item.Sent, formatTime(item.CreatedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.TimeEntries {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO task_time_entries (id, task_id, started_at, ended_at) VALUES (?, ?, ?, ?)`,
			item.ID, item.TaskID, formatTime(item.StartedAt), ptrTimeToNull(item.EndedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.JiraLinks {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO jira_links (task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.TaskID, item.JiraKey, item.JiraURL, item.JiraProjectKey, item.LinkOrigin,
			ptrTimeToNull(item.LastImportedAt), ptrTimeToNull(item.LastExportedAt), ptrToNullString(item.RawStatus),
			formatTime(item.CreatedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.Attachments {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO attachments (id, task_id, note_id, filename, content_type, size_bytes, storage_key, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, ptrToNullInt64(item.TaskID), ptrToNullInt64(item.NoteID), item.Filename, item.ContentType,
			item.SizeBytes, item.StorageKey, formatTime(item.CreatedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.CalendarSources {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO calendar_sources (id, name, url, color, workspace_id, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			item.ID, item.Name, item.URL, item.Color, item.WorkspaceID, formatTime(item.CreatedAt)); err != nil {
			return err
		}
	}
	for _, item := range data.TaskComments {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO task_comments (id, task_id, author, body_md, external_id, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			item.ID, item.TaskID, item.Author, item.BodyMd, ptrToNullString(item.ExternalID), formatTime(item.CreatedAt)); err != nil {
			return err
		}
	}
	if data.AISettings != nil {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO ai_settings (id, provider, base_url, model, api_key, updated_at) VALUES (1, ?, ?, ?, '', ?)`,
			data.AISettings.Provider, data.AISettings.BaseURL, data.AISettings.Model,
			formatTime(data.AISettings.UpdatedAt)); err != nil {
			return err
		}
	}
	if data.Appearance != nil {
		payload, err := json.Marshal(data.Appearance.CustomPalettes)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO appearance_settings (id, global_palette_id, custom_palettes, updated_at) VALUES (1, ?, ?, ?)`,
			data.Appearance.GlobalPaletteID, string(payload), formatTime(data.Appearance.UpdatedAt)); err != nil {
			return err
		}
		for workspace, paletteID := range data.Appearance.WorkspacePalettes {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO workspace_themes (workspace_id, palette_id) VALUES (?, ?)`,
				workspace, paletteID); err != nil {
				return err
			}
		}
	}

	tables := []string{
		"workspaces", "boards", "board_columns", "tasks", "tags", "notes", "reminders",
		"task_time_entries", "attachments", "calendar_sources", "meeting_outcomes", "task_comments",
	}
	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, `DELETE FROM sqlite_sequence WHERE name = ?`, table); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			fmt.Sprintf(`INSERT INTO sqlite_sequence (name, seq) SELECT ?, COALESCE(max(id), 0) FROM %s`, table), table); err != nil {
			return err
		}
	}

	return tx.Commit()
}
