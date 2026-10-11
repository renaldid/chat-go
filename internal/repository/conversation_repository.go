package repository

import (
	"context"

	"github.com/renaldid/chat-go/internal/model"
)

type ConversationRepository interface {
	Create(ctx context.Context, conversation *model.Conversation) error
	GetByID(ctx context.Context, id int64) (*model.Conversation, error)
}
