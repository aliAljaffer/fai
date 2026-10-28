package lite

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func toWorkspace(id int64, name, tag, createdAt string) store.Workspace {
	t, _ := parseTime(createdAt)
	return store.Workspace{ID: id, Name: name, Tag: tag, CreatedAt: t}
}

const workspaceColumns = `id, name, tag, created_at`

func (s *Store) ListWorkspaces(ctx context.Context) ([]store.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+workspaceColumns+` FROM workspaces ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.Workspace{}
	for rows.Next() {
		var w store.Workspace
		var createdAt string
		if err := rows.Scan(&w.ID, &w.Name, &w.Tag, &createdAt); err != nil {
			return nil, err
		}
		w.CreatedAt, _ = parseTime(createdAt)
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) GetWorkspace(ctx context.Context, id int64) (store.Workspace, error) {
	var w store.Workspace
	var createdAt string
	err := s.db.QueryRowContext(ctx, `SELECT `+workspaceColumns+` FROM workspaces WHERE id = ?`, id).
		Scan(&w.ID, &w.Name, &w.Tag, &createdAt)
	if err != nil {
		return store.Workspace{}, err
	}
	w.CreatedAt, _ = parseTime(createdAt)
	return w, nil
}

func (s *Store) CreateWorkspace(ctx context.Context, name, tag string) (store.Workspace, error) {
	tag = strings.ToUpper(tag)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Workspace{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_tag_aliases WHERE upper(tag) = ?`, tag); err != nil {
		return store.Workspace{}, err
	}
	var w store.Workspace
	var createdAt string
	err = tx.QueryRowContext(ctx, `INSERT INTO workspaces (name, tag) VALUES (?, ?)
		ON CONFLICT DO NOTHING
		RETURNING `+workspaceColumns, name, tag).
		Scan(&w.ID, &w.Name, &w.Tag, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return store.Workspace{}, store.ErrDuplicateTag
	}
	if err != nil {
		return store.Workspace{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.Workspace{}, err
	}
	w.CreatedAt, _ = parseTime(createdAt)
	return w, nil
}

func (s *Store) UpdateWorkspace(ctx context.Context, id int64, name, tag *string) (store.Workspace, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Workspace{}, err
	}
	defer tx.Rollback()

	var current store.Workspace
	var createdAt string
	err = tx.QueryRowContext(ctx, `SELECT `+workspaceColumns+` FROM workspaces WHERE id = ?`, id).
		Scan(&current.ID, &current.Name, &current.Tag, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
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
		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM workspaces WHERE upper(tag) = ? AND id != ?`, nextTag, id).Scan(&taken)
		if err != nil {
			return store.Workspace{}, err
		}
		if taken > 0 {
			return store.Workspace{}, store.ErrDuplicateTag
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_tag_aliases WHERE upper(tag) = ?`, nextTag); err != nil {
			return store.Workspace{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_tag_aliases (workspace_id, tag) VALUES (?, ?)
			ON CONFLICT DO NOTHING`, id, current.Tag); err != nil {
			return store.Workspace{}, err
		}
	}

	var w store.Workspace
	err = tx.QueryRowContext(ctx, `UPDATE workspaces SET name = ?, tag = ? WHERE id = ?
		RETURNING `+workspaceColumns, nextName, nextTag, id).
		Scan(&w.ID, &w.Name, &w.Tag, &createdAt)
	if err != nil {
		return store.Workspace{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.Workspace{}, err
	}
	w.CreatedAt, _ = parseTime(createdAt)
	return w, nil
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

	scope := ptrToNullInt64(workspaceID)
	pattern := "%" + store.EscapeLike(query) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.title, t.ref_num, w.tag, w.name
		FROM tasks t
		JOIN workspaces w ON w.id = t.workspace_id
		WHERE t.archived = 0
		  AND (lower(t.title) LIKE lower(?) ESCAPE '\' OR lower(t.description_md) LIKE lower(?) ESCAPE '\')
		  AND (CAST(? AS INTEGER) IS NULL OR t.workspace_id = ?)
		ORDER BY t.updated_at DESC
		LIMIT 10`, pattern, pattern, scope, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskRefs(rows)
}

func (s *Store) resolveRef(ctx context.Context, tag string, num int64, workspaceID *int64) (*store.TaskRef, error) {
	scope := ptrToNullInt64(workspaceID)
	var id, refNum int64
	var title, workspaceTag, workspaceName string
	err := s.db.QueryRowContext(ctx, `
		SELECT t.id, t.title, t.ref_num, w.tag, w.name
		FROM tasks t
		JOIN workspaces w ON w.id = t.workspace_id
		WHERE t.archived = 0 AND t.ref_num = ?
		  AND (upper(w.tag) = ? OR w.id IN (
		    SELECT workspace_id FROM workspace_tag_aliases WHERE upper(tag) = ?
		  ))
		  AND (CAST(? AS INTEGER) IS NULL OR t.workspace_id = ?)
		LIMIT 1`, num, tag, tag, scope, scope).
		Scan(&id, &title, &refNum, &workspaceTag, &workspaceName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &store.TaskRef{
		Kind: "task", ID: id, Title: title, RefNum: refNum,
		Ref:          workspaceTag + "-" + strconv.FormatInt(refNum, 10),
		WorkspaceTag: workspaceTag, WorkspaceName: workspaceName,
	}, nil
}

func scanTaskRefs(rows *sql.Rows) ([]store.TaskRef, error) {
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
	err := s.db.QueryRowContext(ctx, `SELECT workspace_id, ref_num FROM tasks WHERE id = ?`, taskID).
		Scan(&workspaceID, &refNum)
	if errors.Is(err, sql.ErrNoRows) {
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
	rows, err := s.db.QueryContext(ctx, `
		SELECT tag FROM workspaces WHERE id = ?
		UNION
		SELECT tag FROM workspace_tag_aliases WHERE workspace_id = ?`, workspaceID, workspaceID)
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

	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.title, w.tag
		FROM tasks t JOIN workspaces w ON w.id = t.workspace_id
		WHERE t.archived = 0 AND t.id != ? AND t.description_md LIKE ? ESCAPE '\'`, taskID, needle)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b store.Backlink
		b.Kind = "task"
		if err := rows.Scan(&b.ID, &b.Title, &b.WorkspaceTag); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	noteRows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.title, w.tag
		FROM notes n JOIN workspaces w ON w.id = n.workspace_id
		WHERE n.body_md LIKE ? ESCAPE '\' OR n.title LIKE ? ESCAPE '\'`, needle, needle)
	if err != nil {
		return nil, err
	}
	defer noteRows.Close()
	for noteRows.Next() {
		var b store.Backlink
		b.Kind = "note"
		if err := noteRows.Scan(&b.ID, &b.Title, &b.WorkspaceTag); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := noteRows.Err(); err != nil {
		return nil, err
	}
	noteRows.Close()

	reminderRows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.title, w.tag
		FROM reminders r JOIN workspaces w ON w.id = r.workspace_id
		WHERE r.title LIKE ? ESCAPE '\'`, needle)
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
