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
			user_id,
			last_read_message_id
		)
		VALUES (
			$1,
			$2,
			(
				SELECT MAX(id)
				FROM messages
				WHERE conversation_id = $1
			)
		)
		RETURNING joined_at, last_read_message_id
	`

	err := r.db.QueryRow(
		ctx,
		query,
		member.ConversationID,
		member.UserID,
	).Scan(
		&member.JoinedAt,
		&member.LastReadMessageID,
	)
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

func (r *ConversationMemberRepositoryPostgres) MarkRead(
	ctx context.Context,
	conversationID int64,
	userID int64,
	messageID int64,
) (int64, error) {
	query := `
		UPDATE conversation_members
		SET last_read_message_id = CASE
			WHEN last_read_message_id IS NULL OR last_read_message_id < $3
			THEN $3
			ELSE last_read_message_id
		END
		WHERE conversation_id = $1
		  AND user_id = $2
		RETURNING last_read_message_id
	`

	var lastReadMessageID int64

	if err := r.db.QueryRow(
		ctx,
		query,
		conversationID,
		userID,
		messageID,
	).Scan(&lastReadMessageID); err != nil {
		return 0, fmt.Errorf("mark conversation read: %w", err)
	}

	return lastReadMessageID, nil
}
