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

-- name: UpdateJiraLinkImported :exec
UPDATE jira_links SET last_imported_at = now(), raw_status = $2 WHERE task_id = $1;

-- name: DeleteJiraLink :exec
DELETE FROM jira_links WHERE task_id = $1;
