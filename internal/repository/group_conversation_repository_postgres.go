package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renaldid/chat-go/internal/model"
)

type GroupConversationRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewGroupConversationRepositoryPostgres(
	db *pgxpool.Pool,
) *GroupConversationRepositoryPostgres {
	return &GroupConversationRepositoryPostgres{db: db}
}

func (r *GroupConversationRepositoryPostgres) Create(
	ctx context.Context,
	conversation *model.Conversation,
	creatorID int64,
	memberIDs []int64,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create group transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	err = tx.QueryRow(
		ctx,
		`
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
		`,
		model.ConversationTypeGroup,
		conversation.Title,
	).Scan(
		&conversation.ID,
		&conversation.Type,
		&conversation.Title,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create group conversation: %w", err)
	}

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO conversation_members (
				conversation_id,
				user_id,
				role
			)
			VALUES ($1, $2, $3)
		`,
		conversation.ID,
		creatorID,
		model.ConversationRoleOwner,
	)
	if err != nil {
		return fmt.Errorf("add group owner: %w", err)
	}

	for _, memberID := range memberIDs {
		_, err = tx.Exec(
			ctx,
			`
				INSERT INTO conversation_members (
					conversation_id,
					user_id,
					role
				)
				VALUES ($1, $2, $3)
			`,
			conversation.ID,
			memberID,
			model.ConversationRoleMember,
		)
		if err != nil {
			return fmt.Errorf("add group member: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create group transaction: %w", err)
	}

	return nil
}

func (r *GroupConversationRepositoryPostgres) Rename(
	ctx context.Context,
	conversationID int64,
	title string,
) (*model.Conversation, error) {
	conversation := &model.Conversation{}

	err := r.db.QueryRow(
		ctx,
		`
			UPDATE conversations
			SET
				title = $2,
				updated_at = NOW()
			WHERE id = $1
			RETURNING
				id,
				type,
				title,
				created_at,
				updated_at
		`,
		conversationID,
		title,
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
		return nil, fmt.Errorf("rename group conversation: %w", err)
	}

	return conversation, nil
}

func (r *GroupConversationRepositoryPostgres) Leave(
	ctx context.Context,
	conversationID int64,
	userID int64,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin leave group transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var role string

	err = tx.QueryRow(
		ctx,
		`
			SELECT role
			FROM conversation_members
			WHERE conversation_id = $1
			  AND user_id = $2
			FOR UPDATE
		`,
		conversationID,
		userID,
	).Scan(&role)
	if err != nil {
		return fmt.Errorf("get leaving member: %w", err)
	}

	_, err = tx.Exec(
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
		return fmt.Errorf("remove leaving member: %w", err)
	}

	if role == model.ConversationRoleOwner {
		var successorID int64

		err = tx.QueryRow(
			ctx,
			`
				SELECT user_id
				FROM conversation_members
				WHERE conversation_id = $1
				ORDER BY
					CASE role
						WHEN 'admin' THEN 0
						ELSE 1
					END,
					joined_at,
					user_id
				LIMIT 1
			`,
			conversationID,
		).Scan(&successorID)

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			_, err = tx.Exec(
				ctx,
				`
					DELETE FROM conversations
					WHERE id = $1
				`,
				conversationID,
			)
			if err != nil {
				return fmt.Errorf("delete empty group: %w", err)
			}

		case err != nil:
			return fmt.Errorf("find group owner successor: %w", err)

		default:
			_, err = tx.Exec(
				ctx,
				`
					UPDATE conversation_members
					SET role = $3
					WHERE conversation_id = $1
					  AND user_id = $2
				`,
				conversationID,
				successorID,
				model.ConversationRoleOwner,
			)
			if err != nil {
				return fmt.Errorf("transfer group ownership: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit leave group transaction: %w", err)
	}

	return nil
}
