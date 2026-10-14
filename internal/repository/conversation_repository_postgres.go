package repository

import (
	"context"
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
			c.id,
			c.type,
			c.title,
			c.created_at,
			c.updated_at,
			(
				SELECT COUNT(*)
				FROM messages m
				WHERE m.conversation_id = c.id
				  AND m.id > COALESCE(
					  cm.last_read_message_id,
					  0
				  )
				  AND m.sender_id <> cm.user_id
			) AS unread_count
		FROM conversations c
		INNER JOIN conversation_members cm
			ON cm.conversation_id = c.id
		WHERE cm.user_id = $1
		ORDER BY
			c.updated_at DESC,
			c.id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := r.db.Query(
		ctx,
		query,
		userID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list conversations by user id: %w",
			err,
		)
	}
	defer rows.Close()

	conversations := make([]model.ConversationSummary, 0)
	for rows.Next() {
		var conversation model.ConversationSummary

		if err := rows.Scan(
			&conversation.ID,
			&conversation.Type,
			&conversation.Title,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
			&conversation.UnreadCount,
		); err != nil {
			return nil, fmt.Errorf(
				"scan conversation: %w",
				err,
			)
		}

		conversations = append(
			conversations,
			conversation,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate conversations: %w",
			err,
		)
	}

	return conversations, nil
}
