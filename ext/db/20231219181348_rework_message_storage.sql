-- +goose Up
DROP TABLE messages;

CREATE TABLE messages (
    message_id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deadletter JSONB NOT NULL
);

-- +goose Down
SELECT 1;
