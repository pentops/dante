-- +goose Up
CREATE TABLE deadmessage (
    message_id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    state JSONB NOT NULL
);

CREATE TABLE deadmessage_event (
    id UUID PRIMARY KEY,
    message_id UUID REFERENCES deadmessage(message_id) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    sequence int NOT NULL,
    cause jsonb NOT NULL,
	  data jsonb NOT NULL
);

CREATE INDEX message_id_idx ON deadmessage_event(message_id);

-- +goose Down
DROP TABLE messages;
DROP TABLE deadmessage;
