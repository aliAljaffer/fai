-- name: GetJiraLinkByKey :one
SELECT task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at
FROM jira_links WHERE jira_key = ?;

-- name: GetJiraLinkByTask :one
SELECT task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at
FROM jira_links WHERE task_id = ?;

-- name: CreateJiraLink :one
INSERT INTO jira_links (task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, raw_status)
VALUES (sqlc.arg(task_id), sqlc.arg(jira_key), sqlc.arg(jira_url), sqlc.arg(jira_project_key), sqlc.arg(link_origin), sqlc.arg(last_imported_at), sqlc.arg(raw_status))
RETURNING task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at;

-- name: CreateJiraLinkManual :one
INSERT INTO jira_links (task_id, jira_key, jira_url, jira_project_key, link_origin, raw_status)
VALUES (sqlc.arg(task_id), sqlc.arg(jira_key), sqlc.arg(jira_url), sqlc.arg(jira_project_key), 'manual', sqlc.arg(raw_status))
RETURNING task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at;

-- name: CreateJiraLinkExported :one
INSERT INTO jira_links (task_id, jira_key, jira_url, jira_project_key, link_origin, last_exported_at)
VALUES (sqlc.arg(task_id), sqlc.arg(jira_key), sqlc.arg(jira_url), sqlc.arg(jira_project_key), 'exported', sqlc.arg(last_exported_at))
RETURNING task_id, jira_key, jira_url, jira_project_key, link_origin, last_imported_at, last_exported_at, raw_status, created_at;

-- name: UpdateJiraLinkImported :exec
UPDATE jira_links SET last_imported_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), raw_status = sqlc.arg(raw_status) WHERE task_id = sqlc.arg(task_id);

-- name: DeleteJiraLink :exec
DELETE FROM jira_links WHERE task_id = ?;
