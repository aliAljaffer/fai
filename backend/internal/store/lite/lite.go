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

	return &Store{q: sqlitedb.New(db), db: db}, nil
}

func (s *Store) Close() {
	s.db.Close()
}
