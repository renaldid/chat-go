package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renaldid/chat-go/internal/model"
)

type ConversationMemberRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewConversationMemberRepositoryPostgres(
	db *pgxpool.Pool,
) *ConversationMemberRepositoryPostgres {
	return &ConversationMemberRepositoryPostgres{
		db: db,
	}
}

func (r *ConversationMemberRepositoryPostgres) Add(
	ctx context.Context,
	member *model.ConversationMember,
) error {
	query := `
		INSERT INTO conversation_members (
			conversation_id,
			user_id
		)
		VALUES ($1, $2)
		RETURNING joined_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		member.ConversationID,
		member.UserID,
	).Scan(&member.JoinedAt)
	if err != nil {
		return fmt.Errorf("add conversation member: %w", err)
	}

	return nil
}

func (r *ConversationMemberRepositoryPostgres) Exists(
	ctx context.Context,
	conversationID int64,
	userID int64,
) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM conversation_members
			WHERE conversation_id = $1
			  AND user_id = $2
		)
	`

	var exists bool

	if err := r.db.QueryRow(
		ctx,
		query,
		conversationID,
		userID,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check conversation member: %w", err)
	}

	return exists, nil
}
