package service

import (
	"context"
	"errors"

	"github.com/renaldid/chat-go/internal/model"
	"github.com/renaldid/chat-go/internal/repository"
)

var (
	ErrMessageNotFound          = errors.New("message not found")
	ErrMessageNotInConversation = errors.New("message does not belong to conversation")
)

type ReadReceiptService struct {
	conversationRepository       repository.ConversationRepository
	conversationMemberRepository repository.ConversationMemberRepository
	messageRepository            repository.MessageRepository
}

func NewReadReceiptService(
	conversationRepository repository.ConversationRepository,
	conversationMemberRepository repository.ConversationMemberRepository,
	messageRepository repository.MessageRepository,
) *ReadReceiptService {
	return &ReadReceiptService{
		conversationRepository:       conversationRepository,
		conversationMemberRepository: conversationMemberRepository,
		messageRepository:            messageRepository,
	}
}

func (s *ReadReceiptService) MarkRead(
	ctx context.Context,
	conversationID int64,
	userID int64,
	messageID int64,
) (*model.ReadReceipt, error) {
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

	message, err := s.messageRepository.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, ErrMessageNotFound
	}

	if message.ConversationID != conversationID {
		return nil, ErrMessageNotInConversation
	}

	lastReadMessageID, err := s.conversationMemberRepository.MarkRead(
		ctx,
		conversationID,
		userID,
		messageID,
	)
	if err != nil {
		return nil, err
	}

	return &model.ReadReceipt{
		ConversationID:    conversationID,
		UserID:            userID,
		LastReadMessageID: lastReadMessageID,
	}, nil
}
