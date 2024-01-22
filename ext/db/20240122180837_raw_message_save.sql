-- +goose Up
ALTER TABLE messages ADD COLUMN raw_msg BYTEA;

-- +goose Down
ALTER TABLE messages DROP COLUMN raw_msg;
