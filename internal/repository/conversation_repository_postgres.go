package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renaldid/chat-go/internal/model"
)

type ConversationRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewConversationRepositoryPostgres(db *pgxpool.Pool) *ConversationRepositoryPostgres {
	return &ConversationRepositoryPostgres{db: db}
}

func (r *ConversationRepositoryPostgres) Create(ctx context.Context, conversation *model.Conversation) error {
	conversationType := conversation.Type
	if conversationType == "" {
		conversationType = model.ConversationTypeGroup
	}

	query := `
		INSERT INTO conversations (
			type,
			title
		)
		VALUES ($1, $2)
		RETURNING
			id,
			type,
			title,
			created_at,
			updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		conversationType,
		conversation.Title,
	).Scan(
		&conversation.ID,
		&conversation.Type,
		&conversation.Title,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"create conversation: %w",
			err,
		)
	}

	return nil
}

func (r *ConversationRepositoryPostgres) GetByID(ctx context.Context, id int64) (*model.Conversation, error) {
	query := `
		SELECT
			id,
			type,
			title,
			created_at,
			updated_at
		FROM conversations
		WHERE id = $1
	`

	conversation := &model.Conversation{}

	err := r.db.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&conversation.ID,
		&conversation.Type,
		&conversation.Title,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"get conversation by id: %w",
			err,
		)
	}

	return conversation, nil
}

func (r *ConversationRepositoryPostgres) ListByUserID(ctx context.Context, userID int64, limit int, offset int) ([]model.ConversationSummary, error) {
	query := `
		SELECT
			c.id, c.type, c.title,
			CASE WHEN c.type = 'direct'
				THEN COALESCE(du.display_name, du.username, 'Direct Chat')
				ELSE c.title
			END AS display_title,
			c.created_at, c.updated_at,
			(
				SELECT COUNT(*)
				FROM messages m
				WHERE m.conversation_id = c.id
				  AND m.id > COALESCE(cm.last_read_message_id, 0)
				  AND m.sender_id <> cm.user_id
			) AS unread_count,
			(
				SELECT COUNT(*)
				FROM conversation_members cm2
				WHERE cm2.conversation_id = c.id
			) AS member_count,
			du.id, du.username, du.display_name,
			lm.id, lm.sender_id, lm.sender_username, lm.sender_display_name,
			lm.content, lm.created_at, lm.edited_at, lm.deleted_at
		FROM conversations c
		INNER JOIN conversation_members cm ON cm.conversation_id = c.id
		LEFT JOIN direct_conversations dc ON dc.conversation_id = c.id AND c.type = 'direct'
		LEFT JOIN users du ON du.id = CASE
			WHEN dc.user_low_id = $1 THEN dc.user_high_id
			WHEN dc.user_high_id = $1 THEN dc.user_low_id
		END
		LEFT JOIN LATERAL (
			SELECT m.id, m.sender_id,
				u.username AS sender_username,
				u.display_name AS sender_display_name,
				CASE WHEN m.deleted_at IS NULL THEN m.content ELSE '' END AS content,
				m.created_at, m.edited_at, m.deleted_at
			FROM messages m
			INNER JOIN users u ON u.id = m.sender_id
			WHERE m.conversation_id = c.id
			ORDER BY m.id DESC
			LIMIT 1
		) lm ON TRUE
		WHERE cm.user_id = $1
		ORDER BY c.updated_at DESC, c.id DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list conversations by user id: %w", err)
	}
	defer rows.Close()

	conversations := make([]model.ConversationSummary, 0)

	for rows.Next() {
		var conversation model.ConversationSummary
		var directUserID sql.NullInt64
		var directUsername, directDisplayName sql.NullString
		var lastMessageID, lastSenderID sql.NullInt64
		var lastSenderUsername, lastSenderDisplayName, lastMessageContent sql.NullString
		var lastMessageCreatedAt, lastMessageEditedAt, lastMessageDeletedAt sql.NullTime

		if err := rows.Scan(
			&conversation.ID,
			&conversation.Type,
			&conversation.Title,
			&conversation.DisplayTitle,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
			&conversation.UnreadCount,
			&conversation.MemberCount,
			&directUserID,
			&directUsername,
			&directDisplayName,
			&lastMessageID,
			&lastSenderID,
			&lastSenderUsername,
			&lastSenderDisplayName,
			&lastMessageContent,
			&lastMessageCreatedAt,
			&lastMessageEditedAt,
			&lastMessageDeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}

		if directUserID.Valid {
			conversation.DirectUser = &model.ConversationUserSummary{
				ID:          directUserID.Int64,
				Username:    directUsername.String,
				DisplayName: directDisplayName.String,
			}
		}

		if lastMessageID.Valid {
			lastMessage := &model.ConversationLastMessage{
				ID: lastMessageID.Int64,
				Sender: model.ConversationUserSummary{
					ID:          lastSenderID.Int64,
					Username:    lastSenderUsername.String,
					DisplayName: lastSenderDisplayName.String,
				},
				Content:   lastMessageContent.String,
				CreatedAt: lastMessageCreatedAt.Time,
			}

			if lastMessageEditedAt.Valid {
				editedAt := lastMessageEditedAt.Time
				lastMessage.EditedAt = &editedAt
			}

			if lastMessageDeletedAt.Valid {
				deletedAt := lastMessageDeletedAt.Time
				lastMessage.DeletedAt = &deletedAt
			}

			conversation.LastMessage = lastMessage
		}

		conversations = append(conversations, conversation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}

	return conversations, nil
}
