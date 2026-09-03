
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
-- GishFlow: per-recipient phone number, for the SMS delivery channel
ALTER TABLE targets ADD COLUMN phone varchar(255) DEFAULT '';
ALTER TABLE results ADD COLUMN phone varchar(255) DEFAULT '';
ALTER TABLE email_requests ADD COLUMN phone varchar(255) DEFAULT '';

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
