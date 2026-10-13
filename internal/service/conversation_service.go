package service

import (
	"context"
	"errors"
	"strings"

	"github.com/renaldid/chat-go/internal/model"
	"github.com/renaldid/chat-go/internal/repository"
)

var (
	ErrConversationTitleRequired = errors.New("conversation title is required")
	ErrConversationNotFound      = errors.New("conversation not found")
	ErrConversationAccessDenied  = errors.New("conversation access denied")
)

type ConversationService struct {
	conversationRepository       repository.ConversationRepository
	conversationMemberRepository repository.ConversationMemberRepository
}

func NewConversationService(
	conversationRepository repository.ConversationRepository,
	conversationMemberRepository repository.ConversationMemberRepository,
) *ConversationService {
	return &ConversationService{
		conversationRepository:       conversationRepository,
		conversationMemberRepository: conversationMemberRepository,
	}
}

func (s *ConversationService) Create(ctx context.Context, title string, creatorID int64) (*model.Conversation, error) {
	title = strings.TrimSpace(title)

	if title == "" {
		return nil, ErrConversationTitleRequired
	}

	conversation := &model.Conversation{Title: title}

	if err := s.conversationRepository.Create(ctx, conversation); err != nil {
		return nil, err
	}

	member := &model.ConversationMember{
		ConversationID: conversation.ID,
		UserID:         creatorID,
	}

	if err := s.conversationMemberRepository.Add(ctx, member); err != nil {
		return nil, err
	}

	return conversation, nil
}

func (s *ConversationService) GetByID(ctx context.Context, id int64, userID int64) (*model.Conversation, error) {
	conversation, err := s.conversationRepository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	isMember, err := s.conversationMemberRepository.Exists(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, ErrConversationAccessDenied
	}

	return conversation, nil
}

func (s *ConversationService) ListByUserID(
	ctx context.Context,
	userID int64,
	limit int,
	offset int,
) ([]model.ConversationSummary, error) {
	if limit <= 0 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}

	return s.conversationRepository.ListByUserID(ctx, userID, limit, offset)
}
