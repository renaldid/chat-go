-- +goose Up
-- +goose StatementBegin
ALTER TABLE messages
    ADD COLUMN reply_to_message_id BIGINT NULL
REFERENCES messages(id)
ON DELETE SET NULL;

CREATE INDEX idx_messages_reply_to_message_id
    ON messages(reply_to_message_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_messages_reply_to_message_id;

ALTER TABLE messages
DROP COLUMN IF EXISTS reply_to_message_id;
-- +goose StatementEnd