package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (r *ConversationMemberRepositoryPostgres) Add(ctx context.Context, member *model.ConversationMember) error {
	role := member.Role

	if role == "" {
		role = model.ConversationRoleMember
	}

	query := `
		INSERT INTO conversation_members (
			conversation_id,
			user_id,
			role,
			last_read_message_id
		)
		VALUES (
			$1,
			$2,
			$3,
			(
				SELECT MAX(id)
				FROM messages
				WHERE conversation_id = $1
			)
		)
		RETURNING
			joined_at,
			role,
			last_read_message_id
	`

	err := r.db.QueryRow(
		ctx,
		query,
		member.ConversationID,
		member.UserID,
		role,
	).Scan(
		&member.JoinedAt,
		&member.Role,
		&member.LastReadMessageID,
	)
	if err != nil {
		return fmt.Errorf(
			"add conversation member: %w",
			err,
		)
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

func (r *ConversationMemberRepositoryPostgres) MarkRead(ctx context.Context, conversationID int64, userID int64, messageID int64) (int64, error) {
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

func (r *ConversationMemberRepositoryPostgres) Get(ctx context.Context, conversationID int64, userID int64) (*model.ConversationMember, error) {
	query := `
		SELECT
			conversation_id,
			user_id,
			role,
			joined_at,
			last_read_message_id
		FROM conversation_members
		WHERE conversation_id = $1
		  AND user_id = $2
	`

	member := &model.ConversationMember{}

	err := r.db.QueryRow(
		ctx,
		query,
		conversationID,
		userID,
	).Scan(
		&member.ConversationID,
		&member.UserID,
		&member.Role,
		&member.JoinedAt,
		&member.LastReadMessageID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get conversation member: %w", err)
	}

	return member, nil
}

func (r *ConversationMemberRepositoryPostgres) List(ctx context.Context, conversationID int64) ([]model.ConversationMemberView, error) {
	query := `
		SELECT
			cm.conversation_id,
			cm.user_id,
			u.username,
			u.display_name,
			cm.role,
			cm.joined_at
		FROM conversation_members cm
		INNER JOIN users u
			ON u.id = cm.user_id
		WHERE cm.conversation_id = $1
		ORDER BY
			CASE cm.role
				WHEN 'owner' THEN 0
				WHEN 'admin' THEN 1
				ELSE 2
			END,
			cm.joined_at,
			cm.user_id
	`

	rows, err := r.db.Query(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list conversation members: %w", err)
	}
	defer rows.Close()

	members := make([]model.ConversationMemberView, 0)

	for rows.Next() {
		var member model.ConversationMemberView

		if err := rows.Scan(
			&member.ConversationID,
			&member.UserID,
			&member.Username,
			&member.DisplayName,
			&member.Role,
			&member.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf("scan conversation member: %w", err)
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation members: %w", err)
	}

	return members, nil
}

func (r *ConversationMemberRepositoryPostgres) Remove(ctx context.Context, conversationID int64, userID int64) error {
	_, err := r.db.Exec(
		ctx,
		`
			DELETE FROM conversation_members
			WHERE conversation_id = $1
			  AND user_id = $2
		`,
		conversationID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("remove conversation member: %w", err)
	}

	return nil
}

func (r *ConversationMemberRepositoryPostgres) UpdateRole(ctx context.Context, conversationID int64, userID int64, role string) error {
	_, err := r.db.Exec(
		ctx,
		`
			UPDATE conversation_members
			SET role = $3
			WHERE conversation_id = $1
			  AND user_id = $2
		`,
		conversationID,
		userID,
		role,
	)
	if err != nil {
		return fmt.Errorf("update conversation member role: %w", err)
	}

	return nil
}
