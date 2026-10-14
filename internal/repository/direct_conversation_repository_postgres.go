package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renaldid/chat-go/internal/model"
)

type DirectConversationRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewDirectConversationRepositoryPostgres(
	db *pgxpool.Pool,
) *DirectConversationRepositoryPostgres {
	return &DirectConversationRepositoryPostgres{
		db: db,
	}
}

func (r *DirectConversationRepositoryPostgres) CreateOrGet(ctx context.Context, userID int64, targetUserID int64) (*model.Conversation, bool, error) {
	userLowID, userHighID := normalizeUserPair(userID, targetUserID)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin direct conversation transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	conversation := &model.Conversation{
		Title: "",
	}

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO conversations (
				title
			)
			VALUES ('')
			RETURNING
				id,
				title,
				created_at,
				updated_at
		`,
	).Scan(
		&conversation.ID,
		&conversation.Title,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return nil, false, fmt.Errorf("create direct conversation: %w", err)
	}

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO direct_conversations (
				conversation_id,
				user_low_id,
				user_high_id
			)
			VALUES ($1, $2, $3)
		`,
		conversation.ID,
		userLowID,
		userHighID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				return nil, false, fmt.Errorf("rollback duplicate direct conversation: %w", rollbackErr)
			}

			existingConversation, getErr := r.getByUsers(
				ctx,
				userLowID,
				userHighID,
			)
			if getErr != nil {
				return nil, false, getErr
			}

			return existingConversation, false, nil
		}

		return nil, false, fmt.Errorf("create direct conversation mapping: %w", err)
	}

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO conversation_members (
				conversation_id,
				user_id,
				last_read_message_id
			)
			VALUES
				($1, $2, NULL),
				($1, $3, NULL)
			ON CONFLICT (
				conversation_id,
				user_id
			) DO NOTHING
		`,
		conversation.ID,
		userLowID,
		userHighID,
	)
	if err != nil {
		return nil, false, fmt.Errorf("add direct conversation members: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf(
			"commit direct conversation transaction: %w",
			err,
		)
	}

	return conversation, true, nil
}

func (r *DirectConversationRepositoryPostgres) getByUsers(ctx context.Context, userLowID int64, userHighID int64) (*model.Conversation, error) {
	conversation := &model.Conversation{}

	err := r.db.QueryRow(
		ctx,
		`
			SELECT
				c.id,
				c.title,
				c.created_at,
				c.updated_at
			FROM direct_conversations dc
			INNER JOIN conversations c
				ON c.id = dc.conversation_id
			WHERE dc.user_low_id = $1
			  AND dc.user_high_id = $2
		`,
		userLowID,
		userHighID,
	).Scan(
		&conversation.ID,
		&conversation.Title,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get direct conversation: %w", err)
	}

	return conversation, nil
}

func normalizeUserPair(userID int64, targetUserID int64) (int64, int64) {
	if userID < targetUserID {
		return userID, targetUserID
	}

	return targetUserID, userID
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == "23505"
}
