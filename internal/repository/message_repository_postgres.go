package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renaldid/chat-go/internal/model"
)

type MessageRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewMessageRepositoryPostgres(db *pgxpool.Pool) *MessageRepositoryPostgres {
	return &MessageRepositoryPostgres{db: db}
}

func (r *MessageRepositoryPostgres) Create(ctx context.Context, message *model.Message) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create message transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	query := `
		INSERT INTO messages (
			conversation_id,
			sender_id,
			content
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err = tx.QueryRow(
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

	_, err = tx.Exec(
		ctx,
		`
			UPDATE conversations
			SET updated_at = $1
			WHERE id = $2
		`,
		message.CreatedAt,
		message.ConversationID,
	)
	if err != nil {
		return fmt.Errorf("update conversation activity: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create message transaction: %w", err)
	}

	return nil
}

func (r *MessageRepositoryPostgres) GetByConversationID(
	ctx context.Context,
	conversationID int64,
	beforeID *int64,
	limit int,
) ([]model.Message, error) {
	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
		Err() error
	}

	if beforeID == nil {
		query := `
			SELECT id, conversation_id, sender_id, content, created_at
			FROM (
				SELECT id, conversation_id, sender_id, content, created_at
				FROM messages
				WHERE conversation_id = $1
				ORDER BY id DESC
				LIMIT $2
			) AS result
			ORDER BY id ASC
		`

		result, err := r.db.Query(ctx, query, conversationID, limit)
		if err != nil {
			return nil, fmt.Errorf("get messages by conversation id: %w", err)
		}
		rows = result
	} else {
		query := `
			SELECT id, conversation_id, sender_id, content, created_at
			FROM (
				SELECT id, conversation_id, sender_id, content, created_at
				FROM messages
				WHERE conversation_id = $1
				  AND id < $2
				ORDER BY id DESC
				LIMIT $3
			) AS result
			ORDER BY id ASC
		`

		result, err := r.db.Query(ctx, query, conversationID, *beforeID, limit)
		if err != nil {
			return nil, fmt.Errorf("get messages by conversation id with cursor: %w", err)
		}
		rows = result
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

func (r *MessageRepositoryPostgres) GetByID(
	ctx context.Context,
	id int64,
) (*model.Message, error) {
	query := `
		SELECT id, conversation_id, sender_id, content, created_at
		FROM messages
		WHERE id = $1
	`

	message := &model.Message{}

	err := r.db.QueryRow(ctx, query, id).Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Content,
		&message.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get message by id: %w", err)
	}

	return message, nil
}
