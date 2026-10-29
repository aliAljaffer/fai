-- name: GetJiraLinkByKey :one
SELECT task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at
FROM jira_links WHERE jira_key = $1;

-- name: GetJiraLinkByTask :one
SELECT task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at
FROM jira_links WHERE task_id = $1;

-- name: CreateJiraLink :one
INSERT INTO jira_links (task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, raw_status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at;

-- name: CreateJiraLinkManual :one
INSERT INTO jira_links (task_id, jira_key, jira_url, jira_project_key, link_origin, raw_status)
VALUES ($1, $2, $3, $4, 'manual', $5)
RETURNING task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at;

-- name: CreateJiraLinkExported :one
INSERT INTO jira_links (task_id, jira_key, jira_url, jira_project_key, link_origin, last_exported_at)
VALUES ($1, $2, $3, $4, 'exported', $5)
RETURNING task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at;

-- name: ListJiraLinksByWorkspace :many
SELECT j.task_id, j.jira_key, j.jira_url, j.jira_project_key, j.link_origin, j.last_imported_at, j.last_exported_at, j.raw_status, j.created_at
FROM jira_links j
JOIN tasks t ON t.id = j.task_id
WHERE sqlc.narg('workspace_id')::bigint IS NULL OR t.workspace_id = sqlc.narg('workspace_id')
ORDER BY j.jira_key;

-- name: GetJiraTaskRef :one
SELECT t.id, t.title, t.ref_num, w.tag, w.name
FROM jira_links j
JOIN tasks t ON t.id = j.task_id
JOIN workspaces w ON w.id = t.workspace_id
WHERE j.jira_key = $1;

-- name: UpdateJiraLinkImported :exec
UPDATE jira_links SET last_imported_at = now(), raw_status = $2 WHERE task_id = $1;

-- name: DeleteJiraLink :exec
DELETE FROM jira_links WHERE task_id = $1;
