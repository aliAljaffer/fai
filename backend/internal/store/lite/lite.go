// Package lite implements store.Store against SQLite, used by the docker-compose
// deployment. There's no separate migration-runner step wired into that
// deployment, so New applies the embedded schema itself.
package lite

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	migrationssqlite "github.com/alialjaffer/fai/backend/migrations_sqlite"

	"github.com/alialjaffer/fai/backend/internal/db/sqlitedb"
	"github.com/alialjaffer/fai/backend/internal/store"
)

type Store struct {
	q  *sqlitedb.Queries
	db *sql.DB
}

var _ store.Store = (*Store)(nil)

// New opens (creating if absent) the SQLite file at path and applies the embedded
// schema if it hasn't been applied yet. A single connection is used deliberately —
// SQLite has one writer at a time regardless, this is a personal single-user tool
// with no distributed-system need for a pool, and it sidesteps having to push
// PRAGMA foreign_keys=ON onto every pooled connection individually.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	var exists int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='boards'").Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking schema state: %w", err)
	}
	if exists == 0 {
		if _, err := db.ExecContext(ctx, migrationssqlite.Schema); err != nil {
			return nil, fmt.Errorf("applying sqlite schema: %w", err)
		}
	}
	// Additive migrations are idempotent (IF NOT EXISTS) so existing SQLite
	// files pick them up on upgrade.
	if _, err := db.ExecContext(ctx, migrationssqlite.Attachments); err != nil {
		return nil, fmt.Errorf("applying sqlite attachments schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, migrationssqlite.CalendarSources); err != nil {
		return nil, fmt.Errorf("applying sqlite calendar sources schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, migrationssqlite.CalendarFeedCache); err != nil {
		return nil, fmt.Errorf("applying sqlite calendar feed cache schema: %w", err)
	}
	// SQLite has no ADD COLUMN IF NOT EXISTS, so upgrade columns are checked first.
	columns := []struct{ table, column, definition string }{
		{"boards", "workspace_id", "INTEGER REFERENCES workspaces(id)"},
		{"notes", "workspace_id", "INTEGER REFERENCES workspaces(id)"},
		{"reminders", "workspace_id", "INTEGER REFERENCES workspaces(id)"},
		{"calendar_sources", "workspace_id", "INTEGER REFERENCES workspaces(id)"},
		{"meeting_outcomes", "workspace_id", "INTEGER REFERENCES workspaces(id)"},
		{"tasks", "workspace_id", "INTEGER REFERENCES workspaces(id)"},
		{"tasks", "ref_num", "INTEGER"},
		{"tasks", "kind", "TEXT NOT NULL DEFAULT 'task'"},
	}
	kindAdded := false
	for _, c := range columns {
		added, err := addColumnIfMissing(ctx, db, c.table, c.column, c.definition)
		if err != nil {
			return nil, fmt.Errorf("adding %s.%s: %w", c.table, c.column, err)
		}
		if added && c.table == "tasks" && c.column == "kind" {
			kindAdded = true
		}
	}
	if kindAdded {
		if _, err := db.ExecContext(ctx, `UPDATE tasks SET kind = 'step' WHERE parent_task_id IS NOT NULL`); err != nil {
			return nil, fmt.Errorf("backfilling task kinds: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx, migrationssqlite.Workspaces); err != nil {
		return nil, fmt.Errorf("applying sqlite workspaces schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, migrationssqlite.TaskComments); err != nil {
		return nil, fmt.Errorf("applying sqlite task comments schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, migrationssqlite.AISettings); err != nil {
		return nil, fmt.Errorf("applying sqlite ai settings schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, migrationssqlite.Appearance); err != nil {
		return nil, fmt.Errorf("applying sqlite appearance schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, migrationssqlite.ChatMessages); err != nil {
		return nil, fmt.Errorf("applying sqlite chat messages schema: %w", err)
	}

	return &Store{q: sqlitedb.New(db), db: db}, nil
}

func addColumnIfMissing(ctx context.Context, db *sql.DB, table, column, definition string) (bool, error) {
	var exists int
	err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists > 0 {
		return false, nil
	}
	_, err = db.ExecContext(ctx, "ALTER TABLE "+table+" ADD COLUMN "+column+" "+definition)
	return err == nil, err
}

func (s *Store) Close() {
	s.db.Close()
}
