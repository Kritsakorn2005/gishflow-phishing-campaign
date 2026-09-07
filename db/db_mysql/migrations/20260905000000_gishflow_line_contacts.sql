
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
-- GishFlow: LINE userIds auto-collected from the OA webhook (follow/message)
CREATE TABLE IF NOT EXISTS line_contacts (id INTEGER PRIMARY KEY AUTO_INCREMENT, line_id VARCHAR(255), display_name VARCHAR(255), created_date DATETIME);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE line_contacts;
