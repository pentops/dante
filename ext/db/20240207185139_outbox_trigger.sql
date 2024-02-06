-- +goose Up
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
DROP TRIGGER outbox_notify ON outbox;
DROP FUNCTION outbox_notify;
DROP TABLE outbox;
