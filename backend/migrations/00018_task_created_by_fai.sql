-- +goose Up
ALTER TABLE tasks ADD COLUMN created_by_fai BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE tasks DROP COLUMN created_by_fai;
