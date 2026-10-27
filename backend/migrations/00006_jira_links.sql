-- +goose Up
CREATE TABLE jira_links (
  task_id BIGINT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
  jira_key TEXT NOT NULL UNIQUE,
  jira_url TEXT NOT NULL,
  jira_project_key TEXT NOT NULL,
  link_origin TEXT NOT NULL CHECK (link_origin IN ('imported','exported','manual')),
  last_imported_at TIMESTAMPTZ,
  last_exported_at TIMESTAMPTZ,
  raw_status TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE jira_links;
