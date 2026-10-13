package repository

import (
	"context"

	"github.com/renaldid/chat-go/internal/model"
)

type ConversationMemberRepository interface {
	Add(ctx context.Context, member *model.ConversationMember) error
	Exists(ctx context.Context, conversationID, userID int64) (bool, error)
	MarkRead(ctx context.Context, conversationID, userID, messageID int64) (int64, error)
}
