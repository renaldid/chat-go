-- +goose Up
-- +goose StatementBegin
CREATE TABLE direct_conversations (
                                      conversation_id BIGINT PRIMARY KEY
                                          REFERENCES conversations(id)
                                              ON DELETE CASCADE,

                                      user_low_id BIGINT NOT NULL
                                          REFERENCES users(id)
                                              ON DELETE CASCADE,

                                      user_high_id BIGINT NOT NULL
                                          REFERENCES users(id)
                                              ON DELETE CASCADE,

                                      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

                                      CONSTRAINT chk_direct_conversation_users
                                          CHECK (user_low_id < user_high_id),

                                      CONSTRAINT uq_direct_conversation_users
                                          UNIQUE (user_low_id, user_high_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS direct_conversations;
-- +goose StatementEnd