package pg

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/alialjaffer/fai/backend/internal/store"
)

const workspaceColumns = `id, name, tag, created_at`

func (s *Store) ListWorkspaces(ctx context.Context) ([]store.Workspace, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+workspaceColumns+` FROM workspaces ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.Workspace{}
	for rows.Next() {
		var w store.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Tag, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) GetWorkspace(ctx context.Context, id int64) (store.Workspace, error) {
	var w store.Workspace
	err := s.pool.QueryRow(ctx, `SELECT `+workspaceColumns+` FROM workspaces WHERE id = $1`, id).
		Scan(&w.ID, &w.Name, &w.Tag, &w.CreatedAt)
	return w, err
}

func (s *Store) CreateWorkspace(ctx context.Context, name, tag string) (store.Workspace, error) {
	tag = strings.ToUpper(tag)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return store.Workspace{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM workspace_tag_aliases WHERE upper(tag) = $1`, tag); err != nil {
		return store.Workspace{}, err
	}
	var w store.Workspace
	err = tx.QueryRow(ctx, `INSERT INTO workspaces (name, tag) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
		RETURNING `+workspaceColumns, name, tag).
		Scan(&w.ID, &w.Name, &w.Tag, &w.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Workspace{}, store.ErrDuplicateTag
	}
	if err != nil {
		return store.Workspace{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return store.Workspace{}, err
	}
	return w, nil
}

func (s *Store) UpdateWorkspace(ctx context.Context, id int64, name, tag *string) (store.Workspace, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return store.Workspace{}, err
	}
	defer tx.Rollback(ctx)

	var current store.Workspace
	err = tx.QueryRow(ctx, `SELECT `+workspaceColumns+` FROM workspaces WHERE id = $1`, id).
		Scan(&current.ID, &current.Name, &current.Tag, &current.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Workspace{}, store.ErrNotFound
	}
	if err != nil {
		return store.Workspace{}, err
	}

	nextName, nextTag := current.Name, current.Tag
	if name != nil && *name != "" {
		nextName = *name
	}
	if tag != nil && *tag != "" {
		nextTag = strings.ToUpper(*tag)
	}
	if nextTag != current.Tag {
		var taken int
		err = tx.QueryRow(ctx, `SELECT count(*) FROM workspaces WHERE upper(tag) = $1 AND id != $2`,
			nextTag, id).Scan(&taken)
		if err != nil {
			return store.Workspace{}, err
		}
		if taken > 0 {
			return store.Workspace{}, store.ErrDuplicateTag
		}
		if _, err := tx.Exec(ctx, `DELETE FROM workspace_tag_aliases WHERE upper(tag) = $1`, nextTag); err != nil {
			return store.Workspace{}, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO workspace_tag_aliases (workspace_id, tag) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, id, current.Tag); err != nil {
			return store.Workspace{}, err
		}
	}

	var w store.Workspace
	err = tx.QueryRow(ctx, `UPDATE workspaces SET name = $1, tag = $2 WHERE id = $3
		RETURNING `+workspaceColumns, nextName, nextTag, id).
		Scan(&w.ID, &w.Name, &w.Tag, &w.CreatedAt)
	if err != nil {
		return store.Workspace{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return store.Workspace{}, err
	}
	return w, nil
}

func (s *Store) ResetWorkspaceTasks(ctx context.Context, workspaceID int64) (int64, []string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback(ctx)

	var exists int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM workspaces WHERE id = $1`, workspaceID).Scan(&exists); err != nil {
		return 0, nil, err
	}
	if exists == 0 {
		return 0, nil, store.ErrNotFound
	}

	rows, err := tx.Query(ctx, `
		SELECT a.storage_key
		FROM attachments a
		JOIN tasks t ON t.id = a.task_id
		WHERE t.workspace_id = $1`, workspaceID)
	if err != nil {
		return 0, nil, err
	}
	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return 0, nil, err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, nil, err
	}
	rows.Close()

	tag, err := tx.Exec(ctx, `DELETE FROM tasks WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return 0, nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE workspaces SET ref_seq = 0 WHERE id = $1`, workspaceID); err != nil {
		return 0, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, nil, err
	}
	return tag.RowsAffected(), keys, nil
}

func (s *Store) ResolveRefs(ctx context.Context, query string, workspaceID *int64) ([]store.TaskRef, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []store.TaskRef{}, nil
	}

	if tag, num, ok := store.ParseTaskRef(query); ok {
		ref, err := s.resolveRef(ctx, tag, num, workspaceID)
		if err != nil {
			return nil, err
		}
		if ref != nil {
			return []store.TaskRef{*ref}, nil
		}
	}

	pattern := "%" + store.EscapeLike(query) + "%"
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.title, t.ref_num, w.tag, w.name
		FROM tasks t
		JOIN workspaces w ON w.id = t.workspace_id
		WHERE t.archived = false
		  AND (t.title ILIKE $1 ESCAPE '\' OR t.description_md ILIKE $1 ESCAPE '\')
		  AND ($2::bigint IS NULL OR t.workspace_id = $2)
		ORDER BY t.updated_at DESC
		LIMIT 10`, pattern, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskRefs(rows)
}

func (s *Store) resolveRef(ctx context.Context, tag string, num int64, workspaceID *int64) (*store.TaskRef, error) {
	var ref store.TaskRef
	err := s.pool.QueryRow(ctx, `
		SELECT t.id, t.title, t.ref_num, w.tag, w.name
		FROM tasks t
		JOIN workspaces w ON w.id = t.workspace_id
		WHERE t.archived = false AND t.ref_num = $1
		  AND (upper(w.tag) = $2 OR w.id IN (
		    SELECT workspace_id FROM workspace_tag_aliases WHERE upper(tag) = $2
		  ))
		  AND ($3::bigint IS NULL OR t.workspace_id = $3)
		LIMIT 1`, num, tag, workspaceID).
		Scan(&ref.ID, &ref.Title, &ref.RefNum, &ref.WorkspaceTag, &ref.WorkspaceName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ref.Kind = "task"
	ref.Ref = ref.WorkspaceTag + "-" + strconv.FormatInt(ref.RefNum, 10)
	return &ref, nil
}

func scanTaskRefs(rows pgx.Rows) ([]store.TaskRef, error) {
	out := []store.TaskRef{}
	for rows.Next() {
		var ref store.TaskRef
		if err := rows.Scan(&ref.ID, &ref.Title, &ref.RefNum, &ref.WorkspaceTag, &ref.WorkspaceName); err != nil {
			return nil, err
		}
		ref.Kind = "task"
		ref.Ref = ref.WorkspaceTag + "-" + strconv.FormatInt(ref.RefNum, 10)
		out = append(out, ref)
	}
	return out, rows.Err()
}

// shortcut: LIKE scan over every body; move to a reference table or FTS when
// backlink volume makes the scan slow.
func (s *Store) ListTaskReferences(ctx context.Context, taskID int64) ([]store.Backlink, error) {
	var workspaceID, refNum int64
	err := s.pool.QueryRow(ctx, `SELECT workspace_id, ref_num FROM tasks WHERE id = $1`, taskID).
		Scan(&workspaceID, &refNum)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	tags, err := s.workspaceTags(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	out := []store.Backlink{}
	for _, tag := range tags {
		needle := "%" + store.EscapeLike("@"+tag+"-"+strconv.FormatInt(refNum, 10)) + "%"
		links, err := s.scanBacklinks(ctx, taskID, needle)
		if err != nil {
			return nil, err
		}
		out = append(out, links...)
	}
	return out, nil
}

func (s *Store) workspaceTags(ctx context.Context, workspaceID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT tag FROM workspaces WHERE id = $1
		UNION
		SELECT tag FROM workspace_tag_aliases WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *Store) scanBacklinks(ctx context.Context, taskID int64, needle string) ([]store.Backlink, error) {
	out := []store.Backlink{}

	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.title, w.tag
		FROM tasks t JOIN workspaces w ON w.id = t.workspace_id
		WHERE t.archived = false AND t.id != $1 AND t.description_md ILIKE $2 ESCAPE '\'`, taskID, needle)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var b store.Backlink
		b.Kind = "task"
		if err := rows.Scan(&b.ID, &b.Title, &b.WorkspaceTag); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	noteRows, err := s.pool.Query(ctx, `
		SELECT n.id, n.title, w.tag
		FROM notes n JOIN workspaces w ON w.id = n.workspace_id
		WHERE n.body_md ILIKE $1 ESCAPE '\' OR n.title ILIKE $1 ESCAPE '\'`, needle)
	if err != nil {
		return nil, err
	}
	for noteRows.Next() {
		var b store.Backlink
		b.Kind = "note"
		if err := noteRows.Scan(&b.ID, &b.Title, &b.WorkspaceTag); err != nil {
			noteRows.Close()
			return nil, err
		}
		out = append(out, b)
	}
	if err := noteRows.Err(); err != nil {
		noteRows.Close()
		return nil, err
	}
	noteRows.Close()

	reminderRows, err := s.pool.Query(ctx, `
		SELECT r.id, r.title, w.tag
		FROM reminders r JOIN workspaces w ON w.id = r.workspace_id
		WHERE r.title ILIKE $1 ESCAPE '\'`, needle)
	if err != nil {
		return nil, err
	}
	defer reminderRows.Close()
	for reminderRows.Next() {
		var b store.Backlink
		b.Kind = "reminder"
		if err := reminderRows.Scan(&b.ID, &b.Title, &b.WorkspaceTag); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, reminderRows.Err()
}
