-- Single comprehensive schema for the SQLite (docker-compose) deployment target,
-- mirroring the cumulative state of backend/migrations (Postgres/kubernetes target)
-- as of migration 00008. Translated per-engine: INTEGER PRIMARY KEY AUTOINCREMENT
-- instead of GENERATED ALWAYS AS IDENTITY, TEXT (RFC3339/ISO date strings) instead of
-- TIMESTAMPTZ/DATE, BOOLEAN-as-INTEGER, FTS5 virtual tables + sync triggers instead of
-- tsvector+GIN, julianday() math instead of EXTRACT(EPOCH ...).
PRAGMA foreign_keys = ON;

CREATE TABLE meeting_outcomes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  occurred_on TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  notes_md TEXT NOT NULL DEFAULT '',
  ends_at TEXT,
  outcome_prompt_sent BOOLEAN NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_meeting_outcomes_date ON meeting_outcomes(occurred_on);
CREATE INDEX idx_meeting_outcomes_pending_prompt ON meeting_outcomes(ends_at)
  WHERE ends_at IS NOT NULL AND notes_md = '' AND outcome_prompt_sent = 0;

CREATE TABLE boards (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE board_columns (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  position INTEGER NOT NULL,
  is_done BOOLEAN NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  UNIQUE (board_id, position)
);

CREATE TABLE tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
  column_id INTEGER NOT NULL REFERENCES board_columns(id) ON DELETE RESTRICT,
  parent_task_id INTEGER REFERENCES tasks(id) ON DELETE SET NULL,
  meeting_outcome_id INTEGER REFERENCES meeting_outcomes(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  description_md TEXT NOT NULL DEFAULT '',
  priority TEXT NOT NULL DEFAULT 'medium',
  due_date TEXT,
  position REAL NOT NULL,
  completed_at TEXT,
  archived BOOLEAN NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_tasks_board_column_pos ON tasks(board_id, column_id, position);
CREATE INDEX idx_tasks_completed_at ON tasks(completed_at) WHERE completed_at IS NOT NULL;
CREATE INDEX idx_tasks_parent ON tasks(parent_task_id) WHERE parent_task_id IS NOT NULL;
CREATE INDEX idx_tasks_meeting_outcome ON tasks(meeting_outcome_id) WHERE meeting_outcome_id IS NOT NULL;

CREATE VIRTUAL TABLE tasks_fts USING fts5(title, description_md, content='tasks', content_rowid='id');
CREATE TRIGGER tasks_fts_ai AFTER INSERT ON tasks BEGIN
  INSERT INTO tasks_fts(rowid, title, description_md) VALUES (new.id, new.title, new.description_md);
END;
CREATE TRIGGER tasks_fts_ad AFTER DELETE ON tasks BEGIN
  INSERT INTO tasks_fts(tasks_fts, rowid, title, description_md) VALUES ('delete', old.id, old.title, old.description_md);
END;
CREATE TRIGGER tasks_fts_au AFTER UPDATE ON tasks BEGIN
  INSERT INTO tasks_fts(tasks_fts, rowid, title, description_md) VALUES ('delete', old.id, old.title, old.description_md);
  INSERT INTO tasks_fts(rowid, title, description_md) VALUES (new.id, new.title, new.description_md);
END;

CREATE TABLE tags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  color TEXT NOT NULL DEFAULT '#888888'
);
CREATE TABLE task_tags (
  task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  tag_id  INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (task_id, tag_id)
);

CREATE TABLE notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL UNIQUE,
  body_md TEXT NOT NULL DEFAULT '',
  task_id INTEGER REFERENCES tasks(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE TABLE note_tags (
  note_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  tag_id  INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (note_id, tag_id)
);

CREATE VIRTUAL TABLE notes_fts USING fts5(title, body_md, content='notes', content_rowid='id');
CREATE TRIGGER notes_fts_ai AFTER INSERT ON notes BEGIN
  INSERT INTO notes_fts(rowid, title, body_md) VALUES (new.id, new.title, new.body_md);
END;
CREATE TRIGGER notes_fts_ad AFTER DELETE ON notes BEGIN
  INSERT INTO notes_fts(notes_fts, rowid, title, body_md) VALUES ('delete', old.id, old.title, old.body_md);
END;
CREATE TRIGGER notes_fts_au AFTER UPDATE ON notes BEGIN
  INSERT INTO notes_fts(notes_fts, rowid, title, body_md) VALUES ('delete', old.id, old.title, old.body_md);
  INSERT INTO notes_fts(rowid, title, body_md) VALUES (new.id, new.title, new.body_md);
END;

CREATE TABLE jira_links (
  task_id INTEGER PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
  jira_key TEXT NOT NULL UNIQUE,
  jira_url TEXT NOT NULL,
  jira_project_key TEXT NOT NULL,
  link_origin TEXT NOT NULL CHECK (link_origin IN ('imported','exported','manual')),
  last_imported_at TEXT,
  last_exported_at TEXT,
  raw_status TEXT,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE task_time_entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  ended_at TEXT,
  duration_seconds INTEGER GENERATED ALWAYS AS (
    CASE WHEN ended_at IS NOT NULL THEN CAST((julianday(ended_at) - julianday(started_at)) * 86400 AS INTEGER) END
  ) STORED
);
CREATE INDEX idx_time_entries_task_id ON task_time_entries(task_id);
CREATE UNIQUE INDEX one_running_timer_idx ON task_time_entries ((ended_at IS NULL)) WHERE ended_at IS NULL;

CREATE TABLE reminders (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  remind_at TEXT NOT NULL,
  task_id INTEGER REFERENCES tasks(id) ON DELETE SET NULL,
  sent BOOLEAN NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_reminders_due_unsent ON reminders(remind_at) WHERE sent = 0;

INSERT INTO boards (name) VALUES ('Personal');
INSERT INTO board_columns (board_id, name, position, is_done)
SELECT id, 'To Do', 0, 0 FROM boards WHERE name = 'Personal';
INSERT INTO board_columns (board_id, name, position, is_done)
SELECT id, 'In Progress', 1, 0 FROM boards WHERE name = 'Personal';
INSERT INTO board_columns (board_id, name, position, is_done)
SELECT id, 'Done', 2, 1 FROM boards WHERE name = 'Personal';
