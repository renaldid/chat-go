-- +goose Up
-- +goose StatementBegin

ALTER TABLE conversations
    ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT 'group';

UPDATE conversations
SET type = 'direct'
WHERE id IN (
    SELECT conversation_id
    FROM direct_conversations
);

ALTER TABLE conversations
    ADD CONSTRAINT chk_conversations_type
        CHECK (type IN ('direct', 'group'));

ALTER TABLE conversation_members
    ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'member';

UPDATE conversation_members cm
SET role = 'owner'
WHERE cm.user_id = (
    SELECT cm2.user_id
    FROM conversation_members cm2
    WHERE cm2.conversation_id = cm.conversation_id
    ORDER BY cm2.joined_at ASC, cm2.user_id ASC
    LIMIT 1
    )
  AND EXISTS (
SELECT 1
FROM conversations c
WHERE c.id = cm.conversation_id
  AND c.type = 'group'
    );

ALTER TABLE conversation_members
    ADD CONSTRAINT chk_conversation_member_role
        CHECK (role IN ('owner', 'admin', 'member'));

CREATE UNIQUE INDEX uq_conversation_owner
    ON conversation_members(conversation_id)
    WHERE role = 'owner';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS uq_conversation_owner;

ALTER TABLE conversation_members
DROP CONSTRAINT IF EXISTS chk_conversation_member_role;

ALTER TABLE conversation_members
DROP COLUMN IF EXISTS role;

ALTER TABLE conversations
DROP CONSTRAINT IF EXISTS chk_conversations_type;

ALTER TABLE conversations
DROP COLUMN IF EXISTS type;

-- +goose StatementEnd