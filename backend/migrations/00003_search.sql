-- +goose Up
ALTER TABLE tasks ADD COLUMN search_vector TSVECTOR GENERATED ALWAYS AS (
  to_tsvector('english', coalesce(title,'') || ' ' || coalesce(description_md,''))) STORED;
CREATE INDEX idx_tasks_search ON tasks USING GIN(search_vector);

ALTER TABLE notes ADD COLUMN search_vector TSVECTOR GENERATED ALWAYS AS (
  to_tsvector('english', coalesce(title,'') || ' ' || coalesce(body_md,''))) STORED;
CREATE INDEX idx_notes_search ON notes USING GIN(search_vector);

-- +goose Down
DROP INDEX idx_notes_search;
ALTER TABLE notes DROP COLUMN search_vector;
DROP INDEX idx_tasks_search;
ALTER TABLE tasks DROP COLUMN search_vector;
