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
)

type ConversationService struct {
	conversationRepository repository.ConversationRepository
}

func NewConversationService(
	conversationRepository repository.ConversationRepository,
) *ConversationService {
	return &ConversationService{
		conversationRepository: conversationRepository,
	}
}

func (s *ConversationService) Create(
	ctx context.Context,
	title string,
) (*model.Conversation, error) {
	title = strings.TrimSpace(title)

	if title == "" {
		return nil, ErrConversationTitleRequired
	}

	conversation := &model.Conversation{
		Title: title,
	}

	if err := s.conversationRepository.Create(ctx, conversation); err != nil {
		return nil, err
	}

	return conversation, nil
}

func (s *ConversationService) GetByID(
	ctx context.Context,
	id int64,
) (*model.Conversation, error) {
	conversation, err := s.conversationRepository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	return conversation, nil
}
