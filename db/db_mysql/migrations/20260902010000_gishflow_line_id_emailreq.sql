
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
-- GishFlow: email_requests also embeds BaseRecipient, so it needs line_id too
ALTER TABLE email_requests ADD COLUMN line_id varchar(255) DEFAULT '';

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
