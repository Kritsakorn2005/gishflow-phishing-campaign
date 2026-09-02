
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
-- GishFlow: per-recipient LINE userId, for the LINE delivery channel
ALTER TABLE targets ADD COLUMN line_id varchar(255) DEFAULT '';
ALTER TABLE results ADD COLUMN line_id varchar(255) DEFAULT '';

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
