-- +goose Up
CREATE TABLE boards (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE board_columns (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  board_id BIGINT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  position INTEGER NOT NULL,
  is_done BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (board_id, position)
);

CREATE TABLE tasks (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  board_id BIGINT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
  column_id BIGINT NOT NULL REFERENCES board_columns(id) ON DELETE RESTRICT,
  title TEXT NOT NULL,
  description_md TEXT NOT NULL DEFAULT '',
  priority TEXT NOT NULL DEFAULT 'medium',
  due_date DATE,
  position DOUBLE PRECISION NOT NULL,
  completed_at TIMESTAMPTZ,
  archived BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_tasks_board_column_pos ON tasks(board_id, column_id, position);
CREATE INDEX idx_tasks_completed_at ON tasks(completed_at) WHERE completed_at IS NOT NULL;

CREATE TABLE tags (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  color TEXT NOT NULL DEFAULT '#888888'
);

CREATE TABLE task_tags (
  task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  tag_id  BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (task_id, tag_id)
);

INSERT INTO boards (name) VALUES ('Personal');
INSERT INTO board_columns (board_id, name, position, is_done)
SELECT id, 'To Do', 0, false FROM boards WHERE name = 'Personal';
INSERT INTO board_columns (board_id, name, position, is_done)
SELECT id, 'In Progress', 1, false FROM boards WHERE name = 'Personal';
INSERT INTO board_columns (board_id, name, position, is_done)
SELECT id, 'Done', 2, true FROM boards WHERE name = 'Personal';

-- +goose Down
DROP TABLE task_tags;
DROP TABLE tags;
DROP TABLE tasks;
DROP TABLE board_columns;
DROP TABLE boards;
