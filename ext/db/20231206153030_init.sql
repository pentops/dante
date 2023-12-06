-- +goose Up
CREATE TABLE messages (
    message_id TEXT PRIMARY KEY,
    queue_name TEXT NOT NULL,
    grpc_name TEXT NOT NULL,
    time_of_death TIMESTAMPTZ NOT NULL,
    time_sent TIMESTAMPTZ NOT NULL,
    payload_type TEXT NOT NULL,
    payload_body TEXT NOT NULL,
    error TEXT NOT NULL
);

-- +goose Down
DROP TABLE messages;
