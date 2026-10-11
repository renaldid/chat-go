package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renaldid/chat-go/internal/model"
)

type MessageRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewMessageRepositoryPostgres(
	db *pgxpool.Pool,
) *MessageRepositoryPostgres {
	return &MessageRepositoryPostgres{
		db: db,
	}
}

func (r *MessageRepositoryPostgres) Create(
	ctx context.Context,
	message *model.Message,
) error {
	query := `
		INSERT INTO messages (
			conversation_id,
			sender_id,
			content
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		message.ConversationID,
		message.SenderID,
		message.Content,
	).Scan(
		&message.ID,
		&message.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	return nil
}

func (r *MessageRepositoryPostgres) GetByConversationID(
	ctx context.Context,
	conversationID int64,
	limit int,
	offset int,
) ([]model.Message, error) {
	query := `
		SELECT
			id,
			conversation_id,
			sender_id,
			content,
			created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC, id ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(
		ctx,
		query,
		conversationID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("get messages by conversation id: %w", err)
	}
	defer rows.Close()

	messages := make([]model.Message, 0)

	for rows.Next() {
		var message model.Message

		if err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.Content,
			&message.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	return messages, nil
}
