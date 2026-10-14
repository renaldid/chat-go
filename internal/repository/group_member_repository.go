package repository

import (
	"context"

	"github.com/renaldid/chat-go/internal/model"
)

type GroupMemberRepository interface {
	Get(ctx context.Context, conversationID, userID int64) (*model.ConversationMember, error)
	List(ctx context.Context, conversationID int64) ([]model.ConversationMemberView, error)
	Add(ctx context.Context, member *model.ConversationMember) error
	Remove(ctx context.Context, conversationID, userID int64) error
	UpdateRole(ctx context.Context, conversationID, userID int64, role string) error
}
