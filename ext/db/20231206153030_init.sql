-- +goose Up
CREATE TABLE messages (
    message_id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deadletter JSONB NOT NULL,
    raw_msg JSONB
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

CREATE INDEX message_id_idx ON message_events(message_id);

-- +goose StatementBegin
CREATE FUNCTION outbox_notify()
  RETURNS TRIGGER AS $$ DECLARE
BEGIN
  NOTIFY outboxmessage;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER outbox_notify
AFTER INSERT ON outbox
EXECUTE PROCEDURE outbox_notify();


-- +goose Down
DROP TABLE message_events;
DROP TABLE messages;

DROP TRIGGER outbox_notify ON outbox;
DROP FUNCTION outbox_notify;
DROP TABLE outbox;
