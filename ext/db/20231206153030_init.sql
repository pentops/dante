-- +goose Up
CREATE TABLE messages (
    message_id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deadletter JSONB NOT NULL
);

CREATE TABLE outbox (
    id uuid PRIMARY KEY,
    destination TEXT NOT NULL,
    message BYTEA NOT NULL,
    headers TEXT NOT NULL
);

CREATE TABLE message_events (
    id UUID PRIMARY KEY,
    message_id UUID REFERENCES messages(message_id) NOT NULL,
    tstamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    msg_event JSONB NOT NULL
);

-- +goose Down
DROP TABLE message_events;
DROP TABLE messages;
DROP TABLE outbox;
