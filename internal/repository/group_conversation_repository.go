package repository

import (
	"context"

	"github.com/renaldid/chat-go/internal/model"
)

type GroupConversationRepository interface {
	Create(ctx context.Context, conversation *model.Conversation, creatorID int64, memberIDs []int64) error
	Rename(ctx context.Context, conversationID int64, title string) (*model.Conversation, error)
	Leave(ctx context.Context, conversationID int64, userID int64) error
}
