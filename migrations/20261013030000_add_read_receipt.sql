-- +goose Up
-- +goose StatementBegin
ALTER TABLE conversation_members
ADD COLUMN last_read_message_id BIGINT NULL;

CREATE INDEX idx_messages_conversation_id_id
ON messages(conversation_id, id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_messages_conversation_id_id;

ALTER TABLE conversation_members
DROP COLUMN last_read_message_id;
-- +goose StatementEnd
