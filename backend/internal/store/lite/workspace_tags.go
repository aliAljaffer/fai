package lite

import (
	"context"
	"database/sql"
)

// A tag keeps its id in the lowest workspace that uses it, or in the first
// workspace when nothing uses it. Every other workspace that uses it gets a copy.
const scopeTagsSQL = `
CREATE TABLE tags_scoped (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  workspace_id INTEGER NOT NULL REFERENCES workspaces(id),
  name TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT '#888888',
  UNIQUE (workspace_id, name)
);

CREATE TEMP TABLE tag_usage AS
  SELECT tt.tag_id, k.workspace_id FROM task_tags tt JOIN tasks k ON k.id = tt.task_id
  UNION
  SELECT nt.tag_id, n.workspace_id FROM note_tags nt JOIN notes n ON n.id = nt.note_id;

CREATE TEMP TABLE tag_home AS
  SELECT t.id AS tag_id, COALESCE(
    (SELECT min(u.workspace_id) FROM tag_usage u WHERE u.tag_id = t.id),
    (SELECT min(id) FROM workspaces)) AS workspace_id
  FROM tags t;

INSERT INTO tags_scoped (id, workspace_id, name, color)
SELECT t.id, h.workspace_id, t.name, t.color
FROM tags t JOIN tag_home h ON h.tag_id = t.id;

INSERT INTO tags_scoped (workspace_id, name, color)
SELECT u.workspace_id, t.name, t.color
FROM tag_usage u
JOIN tags t ON t.id = u.tag_id
JOIN tag_home h ON h.tag_id = t.id
WHERE u.workspace_id <> h.workspace_id;

UPDATE task_tags SET tag_id = (
  SELECT s.id FROM tags_scoped s
  JOIN tags t ON t.name = s.name
  JOIN tasks k ON k.workspace_id = s.workspace_id
  WHERE t.id = task_tags.tag_id AND k.id = task_tags.task_id);

UPDATE note_tags SET tag_id = (
  SELECT s.id FROM tags_scoped s
  JOIN tags t ON t.name = s.name
  JOIN notes n ON n.workspace_id = s.workspace_id
  WHERE t.id = note_tags.tag_id AND n.id = note_tags.note_id);

DROP TABLE tag_usage;
DROP TABLE tag_home;
DROP TABLE tags;
ALTER TABLE tags_scoped RENAME TO tags;
`

func scopeTagsToWorkspaces(ctx context.Context, db *sql.DB) error {
	var scoped int
	err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM pragma_table_info('tags') WHERE name = 'workspace_id'`).Scan(&scoped)
	if err != nil || scoped > 0 {
		return err
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// With foreign keys on, DROP TABLE tags deletes every task_tags and
	// note_tags row. The pragma is ignored inside a transaction.
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, "PRAGMA foreign_keys = ON")

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, scopeTagsSQL); err != nil {
		return err
	}
	return tx.Commit()
}
