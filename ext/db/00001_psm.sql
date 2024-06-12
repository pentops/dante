-- +goose Up

CREATE TABLE deadmessage (
  message_id uuid,
  state jsonb NOT NULL,
  CONSTRAINT deadmessage_pk PRIMARY KEY (message_id)
);

ALTER TABLE deadmessage ADD COLUMN description tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(state, '$.data.notification.problem.invariantViolation.description'))) STORED;

CREATE INDEX deadmessage_description_idx ON deadmessage USING GIN (description);

ALTER TABLE deadmessage ADD COLUMN error tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(state, '$.data.notification.problem.unhandledError.error'))) STORED;

CREATE INDEX deadmessage_error_idx ON deadmessage USING GIN (error);

CREATE TABLE deadmessage_event (
  id uuid,
  message_id uuid NOT NULL,
  timestamp timestamptz NOT NULL,
  sequence int NOT NULL,
  data jsonb NOT NULL,
  state jsonb NOT NULL,
  CONSTRAINT deadmessage_event_pk PRIMARY KEY (id),
  CONSTRAINT deadmessage_event_fk_state FOREIGN KEY (message_id) REFERENCES deadmessage(message_id)
);

ALTER TABLE deadmessage_event ADD COLUMN description tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(data, '$.event.created.notification.problem.invariantViolation.urgency'))) STORED;

CREATE INDEX deadmessage_event_description_idx ON deadmessage_event USING GIN (description);

ALTER TABLE deadmessage_event ADD COLUMN error tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(data, '$.event.created.notification.problem.unhandledError.error'))) STORED;

CREATE INDEX deadmessage_event_error_idx ON deadmessage_event USING GIN (error);

-- +goose Down

DROP INDEX deadmessage_event_error_idx;
ALTER TABLE deadmessage_event DROP COLUMN error;

DROP INDEX deadmessage_event_description_idx;
ALTER TABLE deadmessage_event DROP COLUMN description;

DROP TABLE deadmessage_event;

DROP INDEX deadmessage_error_idx;
ALTER TABLE deadmessage DROP COLUMN error;

DROP INDEX deadmessage_description_idx;
ALTER TABLE deadmessage DROP COLUMN description;

DROP TABLE deadmessage;

