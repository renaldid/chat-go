package service

import (
	"context"
	"errors"

	"github.com/renaldid/chat-go/internal/model"
	"github.com/renaldid/chat-go/internal/repository"
)

var ErrDirectConversationWithSelf = errors.New("cannot create direct conversation with self")

type DirectConversationService struct {
	directConversationRepository repository.DirectConversationRepository
	userRepository               repository.UserRepository
}

func NewDirectConversationService(directConversationRepository repository.DirectConversationRepository, userRepository repository.UserRepository) *DirectConversationService {
	return &DirectConversationService{
		directConversationRepository: directConversationRepository,
		userRepository:               userRepository,
	}
}

func (s *DirectConversationService) CreateOrGet(ctx context.Context, userID int64, targetUserID int64) (*model.Conversation, bool, error) {
	if userID == targetUserID {
		return nil, false, ErrDirectConversationWithSelf
	}

	targetUser, err := s.userRepository.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, false, err
	}

	if targetUser == nil {
		return nil, false, ErrUserNotFound
	}

	return s.directConversationRepository.CreateOrGet(
		ctx,
		userID,
		targetUserID,
	)
}
