package service

import (
	"context"
	"errors"

	"github.com/renaldid/chat-go/internal/model"
	"github.com/renaldid/chat-go/internal/repository"
)

var (
	ErrConversationMemberExists = errors.New("user already joined conversation")
)

type ConversationMemberService struct {
	conversationRepository       repository.ConversationRepository
	userRepository               repository.UserRepository
	conversationMemberRepository repository.ConversationMemberRepository
}

func NewConversationMemberService(
	conversationRepository repository.ConversationRepository,
	userRepository repository.UserRepository,
	conversationMemberRepository repository.ConversationMemberRepository,
) *ConversationMemberService {
	return &ConversationMemberService{
		conversationRepository:       conversationRepository,
		userRepository:               userRepository,
		conversationMemberRepository: conversationMemberRepository,
	}
}

func (s *ConversationMemberService) Add(
	ctx context.Context,
	conversationID int64,
	userID int64,
) (*model.ConversationMember, error) {
	conversation, err := s.conversationRepository.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	user, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	exists, err := s.conversationMemberRepository.Exists(
		ctx,
		conversationID,
		userID,
	)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, ErrConversationMemberExists
	}

	member := &model.ConversationMember{
		ConversationID: conversationID,
		UserID:         userID,
	}

	if err := s.conversationMemberRepository.Add(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

func (s *ConversationMemberService) IsMember(ctx context.Context, conversationID int64, userID int64) (bool, error) {
	return s.conversationMemberRepository.Exists(ctx, conversationID, userID)
}
