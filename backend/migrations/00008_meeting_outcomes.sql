-- +goose Up
CREATE TABLE meeting_outcomes (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  occurred_on DATE NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  notes_md TEXT NOT NULL DEFAULT '',
  ends_at TIMESTAMPTZ,
  outcome_prompt_sent BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_meeting_outcomes_date ON meeting_outcomes(occurred_on);
CREATE INDEX idx_meeting_outcomes_pending_prompt ON meeting_outcomes(ends_at)
  WHERE ends_at IS NOT NULL AND notes_md = '' AND outcome_prompt_sent = false;

ALTER TABLE tasks ADD COLUMN meeting_outcome_id BIGINT REFERENCES meeting_outcomes(id) ON DELETE SET NULL;
CREATE INDEX idx_tasks_meeting_outcome ON tasks(meeting_outcome_id) WHERE meeting_outcome_id IS NOT NULL;

-- +goose Down
ALTER TABLE tasks DROP COLUMN meeting_outcome_id;
DROP TABLE meeting_outcomes;
