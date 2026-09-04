
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
-- GishFlow: quiz questions on a training page + per-recipient quiz score
ALTER TABLE training_pages ADD COLUMN quiz text DEFAULT '';
ALTER TABLE results ADD COLUMN quiz_score integer DEFAULT 0;
ALTER TABLE results ADD COLUMN quiz_total integer DEFAULT 0;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
