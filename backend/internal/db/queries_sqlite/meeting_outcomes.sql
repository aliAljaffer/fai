-- name: ListMeetingOutcomes :many
SELECT id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at
FROM meeting_outcomes
WHERE CAST(sqlc.narg('occurred_on') AS TEXT) IS NULL OR occurred_on = sqlc.narg('occurred_on')
ORDER BY occurred_on DESC, created_at DESC;

-- name: ListPendingMeetingOutcomes :many
SELECT id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at
FROM meeting_outcomes
WHERE ends_at IS NOT NULL AND ends_at <= strftime('%Y-%m-%dT%H:%M:%fZ', 'now') AND notes_md = ''
ORDER BY ends_at;

-- name: GetMeetingOutcome :one
SELECT id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at
FROM meeting_outcomes WHERE id = ?;

-- name: CreateMeetingOutcome :one
INSERT INTO meeting_outcomes (occurred_on, title, notes_md, ends_at)
VALUES (sqlc.arg(occurred_on), sqlc.arg(title), sqlc.arg(notes_md), CAST(sqlc.narg('ends_at') AS TEXT))
RETURNING id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at;

-- name: UpdateMeetingOutcome :one
UPDATE meeting_outcomes SET title = sqlc.arg(title), notes_md = sqlc.arg(notes_md), ends_at = CAST(sqlc.narg('ends_at') AS TEXT)
WHERE id = sqlc.arg(id)
RETURNING id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at;

-- name: DeleteMeetingOutcome :exec
DELETE FROM meeting_outcomes WHERE id = ?;

-- name: ListDueUnpromptedMeetingOutcomes :many
SELECT id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at
FROM meeting_outcomes
WHERE ends_at IS NOT NULL AND ends_at <= strftime('%Y-%m-%dT%H:%M:%fZ', 'now') AND notes_md = '' AND outcome_prompt_sent = 0;

-- name: MarkMeetingOutcomePromptSent :exec
UPDATE meeting_outcomes SET outcome_prompt_sent = 1 WHERE id = ?;
