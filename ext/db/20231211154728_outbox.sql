-- +goose Up
CREATE TABLE outbox (
	id uuid PRIMARY KEY,
	destination text NOT NULL,
	message bytea NOT NULL,
	headers text NOT NULL
);

-- +goose Down
DROP TABLE outbox;
