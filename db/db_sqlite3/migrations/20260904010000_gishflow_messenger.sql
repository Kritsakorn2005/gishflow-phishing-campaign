
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
-- GishFlow: per-recipient Messenger PSID, for the Facebook Messenger channel
ALTER TABLE targets ADD COLUMN messenger_id varchar(255) DEFAULT '';
ALTER TABLE results ADD COLUMN messenger_id varchar(255) DEFAULT '';
ALTER TABLE email_requests ADD COLUMN messenger_id varchar(255) DEFAULT '';

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
