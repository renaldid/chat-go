package repository

import (
	"context"

	"github.com/renaldid/chat-go/internal/model"
)

type DirectConversationRepository interface {
	CreateOrGet(ctx context.Context, userID int64, targetUserID int64) (*model.Conversation, bool, error)
}
