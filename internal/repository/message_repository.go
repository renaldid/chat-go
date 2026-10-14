package repository

import (
	"context"

	"github.com/renaldid/chat-go/internal/model"
)

type MessageRepository interface {
	Create(ctx context.Context, message *model.Message) error
	GetByID(ctx context.Context, id int64) (*model.Message, error)
	GetByConversationID(ctx context.Context, conversationID int64, beforeID *int64, limit int) ([]model.Message, error)
	UpdateContent(ctx context.Context, id int64, content string) (*model.Message, error)
	SoftDelete(ctx context.Context, id int64) (*model.Message, error)
}
