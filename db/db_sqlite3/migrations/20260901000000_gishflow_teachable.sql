
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
-- GishFlow: teachable-moment training pages + campaign link + per-result trained flag
CREATE TABLE IF NOT EXISTS "training_pages" ("id" integer primary key autoincrement,"user_id" bigint,"name" varchar(255),"html" varchar(255),"modified_date" datetime );
ALTER TABLE campaigns ADD COLUMN training_id bigint DEFAULT 0;
ALTER TABLE results ADD COLUMN trained boolean DEFAULT 0;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE "training_pages";
