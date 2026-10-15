package service

import (
	"context"
	"errors"
	"strings"

	"github.com/renaldid/chat-go/internal/model"
	"github.com/renaldid/chat-go/internal/repository"
)

var (
	ErrMessageContentRequired        = errors.New("message content is required")
	ErrSenderNotConversationMember   = errors.New("sender is not a conversation member")
	ErrMessagePermissionDenied       = errors.New("message permission denied")
	ErrMessageCannotEditDeleted      = errors.New("deleted message cannot be edited")
	ErrReplyMessageNotFound          = errors.New("reply message not found")
	ErrReplyMessageNotInConversation = errors.New("reply message does not belong to conversation")
	ErrCannotReplyDeletedMessage     = errors.New("cannot reply to deleted message")
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

type MessagePage struct {
	Messages   []model.Message
	NextCursor *int64
	HasMore    bool
}

func (s *MessageService) Create(ctx context.Context, conversationID int64, senderID int64, content string) (*model.Message, error) {
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

func (s *MessageService) GetByConversationID(ctx context.Context, conversationID int64, userID int64, beforeID *int64, limit int) (*MessagePage, error) {
	conversation, err := s.conversationRepository.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	isMember, err := s.conversationMemberRepository.Exists(
		ctx,
		conversationID,
		userID,
	)
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

	messages, err := s.messageRepository.GetByConversationID(
		ctx,
		conversationID,
		beforeID,
		limit+1,
	)
	if err != nil {
		return nil, err
	}

	hasMore := len(messages) > limit

	if hasMore {
		messages = messages[1:]
	}

	var nextCursor *int64

	if hasMore && len(messages) > 0 {
		cursor := messages[0].ID
		nextCursor = &cursor
	}

	return &MessagePage{
		Messages:   messages,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *MessageService) Update(ctx context.Context, conversationID int64, userID int64, messageID int64, content string) (*model.Message, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrMessageContentRequired
	}

	message, err := s.messageForMutation(ctx, conversationID, userID, messageID)
	if err != nil {
		return nil, err
	}
	if message.DeletedAt != nil {
		return nil, ErrMessageCannotEditDeleted
	}

	updated, err := s.messageRepository.UpdateContent(ctx, messageID, content)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrMessageCannotEditDeleted
	}

	return updated, nil
}

func (s *MessageService) Delete(ctx context.Context, conversationID int64, userID int64, messageID int64) (*model.Message, bool, error) {
	message, err := s.messageForMutation(ctx, conversationID, userID, messageID)
	if err != nil {
		return nil, false, err
	}

	if message.DeletedAt != nil {
		return message, false, nil
	}

	deleted, err := s.messageRepository.SoftDelete(ctx, messageID)
	if err != nil {
		return nil, false, err
	}

	if deleted == nil {
		current, err := s.messageRepository.GetByID(ctx, messageID)
		if err != nil {
			return nil, false, err
		}
		if current != nil && current.DeletedAt != nil {
			return current, false, nil
		}
		return nil, false, ErrMessageNotFound
	}

	return deleted, true, nil
}

func (s *MessageService) CreateWithReply(ctx context.Context, conversationID int64, senderID int64, content string, replyToMessageID *int64) (*model.Message, error) {
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

	isMember, err := s.conversationMemberRepository.Exists(ctx, conversationID, senderID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrSenderNotConversationMember
	}

	if replyToMessageID != nil {
		replyMessage, err := s.messageRepository.GetByID(ctx, *replyToMessageID)
		if err != nil {
			return nil, err
		}
		if replyMessage == nil {
			return nil, ErrReplyMessageNotFound
		}
		if replyMessage.ConversationID != conversationID {
			return nil, ErrReplyMessageNotInConversation
		}
		if replyMessage.DeletedAt != nil {
			return nil, ErrCannotReplyDeletedMessage
		}
	}

	message := &model.Message{
		ConversationID:   conversationID,
		SenderID:         senderID,
		Content:          content,
		ReplyToMessageID: replyToMessageID,
	}

	if err := s.messageRepository.Create(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

func (s *MessageService) messageForMutation(ctx context.Context, conversationID int64, userID int64, messageID int64) (*model.Message, error) {
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
	if message.SenderID != userID {
		return nil, ErrMessagePermissionDenied
	}

	return message, nil
}
