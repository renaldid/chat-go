package service

import (
	"context"
	"errors"
	"strings"

	"github.com/renaldid/chat-go/internal/model"
	"github.com/renaldid/chat-go/internal/repository"
)

var (
	ErrMessageContentRequired      = errors.New("message content is required")
	ErrSenderNotConversationMember = errors.New("sender is not a conversation member")
)

type MessageService struct {
	messageRepository            repository.MessageRepository
	conversationRepository       repository.ConversationRepository
	userRepository               repository.UserRepository
	conversationMemberRepository repository.ConversationMemberRepository
}

func NewMessageService(
	messageRepository repository.MessageRepository,
	conversationRepository repository.ConversationRepository,
	userRepository repository.UserRepository,
	conversationMemberRepository repository.ConversationMemberRepository,
) *MessageService {
	return &MessageService{
		messageRepository:            messageRepository,
		conversationRepository:       conversationRepository,
		userRepository:               userRepository,
		conversationMemberRepository: conversationMemberRepository,
	}
}

func (s *MessageService) Create(
	ctx context.Context,
	conversationID int64,
	senderID int64,
	content string,
) (*model.Message, error) {
	content = strings.TrimSpace(content)

	if content == "" {
		return nil, ErrMessageContentRequired
	}

	conversation, err := s.conversationRepository.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	user, err := s.userRepository.GetByID(ctx, senderID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	isMember, err := s.conversationMemberRepository.Exists(
		ctx,
		conversationID,
		senderID,
	)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, ErrSenderNotConversationMember
	}

	message := &model.Message{
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
	}

	if err := s.messageRepository.Create(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

func (s *MessageService) GetByConversationID(
	ctx context.Context,
	conversationID int64,
	userID int64,
	limit int,
	offset int,
) ([]model.Message, error) {
	conversation, err := s.conversationRepository.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	isMember, err := s.conversationMemberRepository.Exists(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, ErrConversationAccessDenied
	}

	if limit <= 0 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}

	return s.messageRepository.GetByConversationID(ctx, conversationID, limit, offset)
}
