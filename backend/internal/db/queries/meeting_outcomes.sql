-- name: ListMeetingOutcomes :many
SELECT id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at, workspace_id,
       calendar_source_id, event_uid, event_title, event_url
FROM meeting_outcomes
WHERE (sqlc.narg('occurred_on')::date IS NULL OR occurred_on = sqlc.narg('occurred_on'))
  AND (sqlc.narg('workspace_id')::bigint IS NULL OR workspace_id = sqlc.narg('workspace_id'))
ORDER BY occurred_on DESC, created_at DESC;

-- name: ListPendingMeetingOutcomes :many
SELECT id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at, workspace_id,
       calendar_source_id, event_uid, event_title, event_url
FROM meeting_outcomes
WHERE ends_at IS NOT NULL AND ends_at <= now() AND notes_md = ''
ORDER BY ends_at;

-- name: GetMeetingOutcome :one
SELECT id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at, workspace_id,
       calendar_source_id, event_uid, event_title, event_url
FROM meeting_outcomes WHERE id = $1;

-- name: GetMeetingOutcomeByEvent :one
SELECT id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at, workspace_id,
       calendar_source_id, event_uid, event_title, event_url
FROM meeting_outcomes
WHERE calendar_source_id = $1 AND event_uid = $2 AND event_uid <> ''
ORDER BY created_at DESC
LIMIT 1;

-- name: CreateMeetingOutcome :one
INSERT INTO meeting_outcomes (occurred_on, title, notes_md, ends_at, workspace_id,
                              calendar_source_id, event_uid, event_title, event_url)
VALUES ($1, $2, $3, $4, COALESCE(NULLIF(sqlc.arg(workspace_id)::bigint, 0), (SELECT id FROM workspaces ORDER BY id LIMIT 1)),
        sqlc.narg('calendar_source_id'), sqlc.arg(event_uid), sqlc.arg(event_title), sqlc.arg(event_url))
RETURNING id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at, workspace_id,
          calendar_source_id, event_uid, event_title, event_url;

-- name: UpdateMeetingOutcome :one
UPDATE meeting_outcomes SET title = $2, notes_md = $3, ends_at = $4
WHERE id = $1
RETURNING id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at, workspace_id,
          calendar_source_id, event_uid, event_title, event_url;

-- name: DeleteMeetingOutcome :exec
DELETE FROM meeting_outcomes WHERE id = $1;

-- name: ListDueUnpromptedMeetingOutcomes :many
SELECT id, occurred_on, title, notes_md, ends_at, outcome_prompt_sent, created_at, workspace_id,
       calendar_source_id, event_uid, event_title, event_url
FROM meeting_outcomes
WHERE ends_at IS NOT NULL AND ends_at <= now() AND notes_md = '' AND outcome_prompt_sent = false;

-- name: MarkMeetingOutcomePromptSent :exec
UPDATE meeting_outcomes SET outcome_prompt_sent = true WHERE id = $1;
