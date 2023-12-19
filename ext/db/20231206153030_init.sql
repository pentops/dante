-- +goose Up
CREATE TABLE messages (
    message_id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deadletter JSONB NOT NULL
);

CREATE TABLE outbox (
    id uuid PRIMARY KEY,
    destination text NOT NULL,
    message bytea NOT NULL,
    headers text NOT NULL
);

-- +goose Down
DROP TABLE messages;
DROP TABLE outbox;
