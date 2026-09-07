
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
-- GishFlow: LINE userIds auto-collected from the OA webhook (follow/message)
CREATE TABLE IF NOT EXISTS "line_contacts" ("id" integer primary key autoincrement,"line_id" varchar(255),"display_name" varchar(255),"created_date" datetime );

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE "line_contacts";
