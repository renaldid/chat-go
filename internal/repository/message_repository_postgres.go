package repository

import (
	"context"
	"database/sql"
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

	err = tx.QueryRow(ctx, `
		INSERT INTO messages (
			conversation_id,
			sender_id,
			content,
			reply_to_message_id
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`,
		message.ConversationID,
		message.SenderID,
		message.Content,
		message.ReplyToMessageID,
	).Scan(
		&message.ID,
		&message.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE conversations
		SET updated_at = $1
		WHERE id = $2
	`, message.CreatedAt, message.ConversationID)
	if err != nil {
		return fmt.Errorf("update conversation activity: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create message transaction: %w", err)
	}

	createdMessage, err := r.GetByID(ctx, message.ID)
	if err != nil {
		return err
	}
	if createdMessage == nil {
		return fmt.Errorf("created message not found")
	}

	*message = *createdMessage
	return nil
}

func (r *MessageRepositoryPostgres) GetByConversationID(ctx context.Context, conversationID int64, beforeID *int64, limit int) ([]model.Message, error) {
	rows, err := r.db.Query(ctx, `
		WITH selected_messages AS (
			SELECT
				id,
				conversation_id,
				sender_id,
				content,
				created_at,
				edited_at,
				deleted_at,
				reply_to_message_id
			FROM messages
			WHERE conversation_id = $1
			  AND ($2::BIGINT IS NULL OR id < $2)
			ORDER BY id DESC
			LIMIT $3
		)
		SELECT
			m.id,
			m.conversation_id,
			m.sender_id,
			CASE WHEN m.deleted_at IS NULL THEN m.content ELSE '' END,
			m.created_at,
			m.edited_at,
			m.deleted_at,
			m.reply_to_message_id,
			rm.id,
			rm.sender_id,
			ru.username,
			ru.display_name,
			CASE WHEN rm.deleted_at IS NULL THEN rm.content ELSE '' END,
			rm.created_at,
			rm.edited_at,
			rm.deleted_at
		FROM selected_messages m
		LEFT JOIN messages rm ON rm.id = m.reply_to_message_id
		LEFT JOIN users ru ON ru.id = rm.sender_id
		ORDER BY m.id ASC
	`, conversationID, beforeID, limit)
	if err != nil {
		return nil, fmt.Errorf("get messages by conversation id: %w", err)
	}
	defer rows.Close()

	messages := make([]model.Message, 0)

	for rows.Next() {
		message, err := scanMessageWithReply(rows)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, *message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	return messages, nil
}

func (r *MessageRepositoryPostgres) GetByID(ctx context.Context, id int64) (*model.Message, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			m.id,
			m.conversation_id,
			m.sender_id,
			CASE WHEN m.deleted_at IS NULL THEN m.content ELSE '' END,
			m.created_at,
			m.edited_at,
			m.deleted_at,
			m.reply_to_message_id,
			rm.id,
			rm.sender_id,
			ru.username,
			ru.display_name,
			CASE WHEN rm.deleted_at IS NULL THEN rm.content ELSE '' END,
			rm.created_at,
			rm.edited_at,
			rm.deleted_at
		FROM messages m
		LEFT JOIN messages rm ON rm.id = m.reply_to_message_id
		LEFT JOIN users ru ON ru.id = rm.sender_id
		WHERE m.id = $1
	`, id)

	message, err := scanMessageWithReply(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get message by id: %w", err)
	}

	return message, nil
}

func (r *MessageRepositoryPostgres) UpdateContent(ctx context.Context, id int64, content string) (*model.Message, error) {
	result, err := r.db.Exec(ctx, `
		UPDATE messages
		SET content = $2, edited_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
	`, id, content)
	if err != nil {
		return nil, fmt.Errorf("update message content: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, nil
	}

	message, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (r *MessageRepositoryPostgres) SoftDelete(ctx context.Context, id int64) (*model.Message, error) {
	result, err := r.db.Exec(ctx, `
		UPDATE messages
		SET deleted_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
	`, id)
	if err != nil {
		return nil, fmt.Errorf("soft delete message: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, nil
	}

	message, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func scanMessageWithReply(scanner interface{ Scan(...any) error }) (*model.Message, error) {
	message := &model.Message{}

	var (
		replyToMessageID       sql.NullInt64
		replyID                sql.NullInt64
		replySenderID          sql.NullInt64
		replySenderUsername    sql.NullString
		replySenderDisplayName sql.NullString
		replyContent           sql.NullString
		replyCreatedAt         sql.NullTime
		replyEditedAt          sql.NullTime
		replyDeletedAt         sql.NullTime
	)

	err := scanner.Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Content,
		&message.CreatedAt,
		&message.EditedAt,
		&message.DeletedAt,
		&replyToMessageID,
		&replyID,
		&replySenderID,
		&replySenderUsername,
		&replySenderDisplayName,
		&replyContent,
		&replyCreatedAt,
		&replyEditedAt,
		&replyDeletedAt,
	)
	if err != nil {
		return nil, err
	}

	if replyToMessageID.Valid {
		id := replyToMessageID.Int64
		message.ReplyToMessageID = &id
	}

	if replyID.Valid {
		reply := &model.MessageReply{
			ID:                replyID.Int64,
			SenderID:          replySenderID.Int64,
			SenderUsername:    replySenderUsername.String,
			SenderDisplayName: replySenderDisplayName.String,
			Content:           replyContent.String,
			CreatedAt:         replyCreatedAt.Time,
		}

		if replyEditedAt.Valid {
			editedAt := replyEditedAt.Time
			reply.EditedAt = &editedAt
		}

		if replyDeletedAt.Valid {
			deletedAt := replyDeletedAt.Time
			reply.DeletedAt = &deletedAt
		}

		message.ReplyTo = reply
	}

	return message, nil
}
