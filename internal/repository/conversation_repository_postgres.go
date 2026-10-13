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
	query := `
		INSERT INTO conversations (title)
		VALUES ($1)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query, conversation.Title).Scan(
		&conversation.ID,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}

	return nil
}

func (r *ConversationRepositoryPostgres) GetByID(ctx context.Context, id int64) (*model.Conversation, error) {
	query := `
		SELECT id, title, created_at, updated_at
		FROM conversations
		WHERE id = $1
	`

	conversation := &model.Conversation{}

	err := r.db.QueryRow(ctx, query, id).Scan(
		&conversation.ID,
		&conversation.Title,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get conversation by id: %w", err)
	}

	return conversation, nil
}

func (r *ConversationRepositoryPostgres) ListByUserID(ctx context.Context, userID int64, limit int, offset int) ([]model.Conversation, error) {
	query := `
		SELECT c.id, c.title, c.created_at, c.updated_at
		FROM conversations c
		INNER JOIN conversation_members cm ON cm.conversation_id = c.id
		WHERE cm.user_id = $1
		ORDER BY c.updated_at DESC, c.id DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list conversations by user id: %w", err)
	}
	defer rows.Close()

	conversations := make([]model.Conversation, 0)

	for rows.Next() {
		var conversation model.Conversation

		if err := rows.Scan(
			&conversation.ID,
			&conversation.Title,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}

		conversations = append(conversations, conversation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}

	return conversations, nil
}
