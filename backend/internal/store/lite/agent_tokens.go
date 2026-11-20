package lite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/alialjaffer/fai/backend/internal/store"
)

const agentTokenColumns = `id, name, prefix, workspace_id, created_at, last_used_at`

func scanAgentToken(row interface{ Scan(...any) error }) (store.AgentToken, error) {
	var t store.AgentToken
	var workspaceID sql.NullInt64
	var createdAt string
	var lastUsedAt sql.NullString
	if err := row.Scan(&t.ID, &t.Name, &t.Prefix, &workspaceID, &createdAt, &lastUsedAt); err != nil {
		return store.AgentToken{}, err
	}
	t.WorkspaceID = nullInt64ToPtr(workspaceID)
	t.CreatedAt, _ = parseTime(createdAt)
	t.LastUsedAt = nullTimeToPtr(lastUsedAt)
	return t, nil
}

func (s *Store) CreateAgentToken(ctx context.Context, name, tokenHash, prefix string, workspaceID *int64) (store.AgentToken, error) {
	var workspace sql.NullInt64
	if workspaceID != nil {
		workspace = sql.NullInt64{Int64: *workspaceID, Valid: true}
	}
	return scanAgentToken(s.db.QueryRowContext(ctx,
		`INSERT INTO agent_tokens (name, token_hash, prefix, workspace_id) VALUES (?, ?, ?, ?)
		 RETURNING `+agentTokenColumns,
		name, tokenHash, prefix, workspace))
}

func (s *Store) ListAgentTokens(ctx context.Context) ([]store.AgentToken, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+agentTokenColumns+` FROM agent_tokens ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.AgentToken{}
	for rows.Next() {
		token, err := scanAgentToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, token)
	}
	return out, rows.Err()
}

func (s *Store) DeleteAgentToken(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM agent_tokens WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *Store) AgentTokenByHash(ctx context.Context, tokenHash string) (store.AgentToken, error) {
	token, err := scanAgentToken(s.db.QueryRowContext(ctx,
		`SELECT `+agentTokenColumns+` FROM agent_tokens WHERE token_hash = ?`, tokenHash))
	if errors.Is(err, sql.ErrNoRows) {
		return store.AgentToken{}, store.ErrNotFound
	}
	return token, err
}

func (s *Store) TouchAgentToken(ctx context.Context, id int64, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE agent_tokens SET last_used_at = ? WHERE id = ?`, formatTime(at), id)
	return err
}
