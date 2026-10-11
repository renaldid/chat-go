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

func NewConversationRepositoryPostgres(
	db *pgxpool.Pool,
) *ConversationRepositoryPostgres {
	return &ConversationRepositoryPostgres{
		db: db,
	}
}

func (r *ConversationRepositoryPostgres) Create(
	ctx context.Context,
	conversation *model.Conversation,
) error {
	query := `
		INSERT INTO conversations (
			title
		)
		VALUES ($1)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		conversation.Title,
	).Scan(
		&conversation.ID,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}

	return nil
}

func (r *ConversationRepositoryPostgres) GetByID(
	ctx context.Context,
	id int64,
) (*model.Conversation, error) {
	query := `
		SELECT
			id,
			title,
			created_at,
			updated_at
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
