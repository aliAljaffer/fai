package pg

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/alialjaffer/fai/backend/internal/store"
)

const agentTokenColumns = `id, name, prefix, workspace_id, created_at, last_used_at`

func (s *Store) CreateAgentToken(ctx context.Context, name, tokenHash, prefix string, workspaceID *int64) (store.AgentToken, error) {
	var t store.AgentToken
	err := s.pool.QueryRow(ctx,
		`INSERT INTO agent_tokens (name, token_hash, prefix, workspace_id) VALUES ($1, $2, $3, $4)
		 RETURNING `+agentTokenColumns,
		name, tokenHash, prefix, workspaceID).
		Scan(&t.ID, &t.Name, &t.Prefix, &t.WorkspaceID, &t.CreatedAt, &t.LastUsedAt)
	return t, err
}

func (s *Store) ListAgentTokens(ctx context.Context) ([]store.AgentToken, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+agentTokenColumns+` FROM agent_tokens ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.AgentToken{}
	for rows.Next() {
		var t store.AgentToken
		if err := rows.Scan(&t.ID, &t.Name, &t.Prefix, &t.WorkspaceID, &t.CreatedAt, &t.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) DeleteAgentToken(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM agent_tokens WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *Store) AgentTokenByHash(ctx context.Context, tokenHash string) (store.AgentToken, error) {
	var t store.AgentToken
	err := s.pool.QueryRow(ctx,
		`SELECT `+agentTokenColumns+` FROM agent_tokens WHERE token_hash = $1`, tokenHash).
		Scan(&t.ID, &t.Name, &t.Prefix, &t.WorkspaceID, &t.CreatedAt, &t.LastUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.AgentToken{}, store.ErrNotFound
	}
	return t, err
}

func (s *Store) TouchAgentToken(ctx context.Context, id int64, at time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE agent_tokens SET last_used_at = $1 WHERE id = $2`, at, id)
	return err
}
