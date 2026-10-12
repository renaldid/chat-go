package service

import (
	"context"
	"errors"
	"testing"

	"github.com/renaldid/chat-go/internal/model"
)

type mockMessageRepository struct {
	messages []model.Message
}

func newMockMessageRepository() *mockMessageRepository {
	return &mockMessageRepository{
		messages: make([]model.Message, 0),
	}
}

func (m *mockMessageRepository) Create(
	ctx context.Context,
	message *model.Message,
) error {
	message.ID = int64(len(m.messages) + 1)

	m.messages = append(m.messages, *message)

	return nil
}

func (m *mockMessageRepository) GetByConversationID(
	ctx context.Context,
	conversationID int64,
	limit int,
	offset int,
) ([]model.Message, error) {
	result := make([]model.Message, 0)

	for _, message := range m.messages {
		if message.ConversationID == conversationID {
			result = append(result, message)
		}
	}

	if offset >= len(result) {
		return []model.Message{}, nil
	}

	end := offset + limit

	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

func TestMessageServiceCreate(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	messageRepo := newMockMessageRepository()

	userService := NewUserService(userRepo)
	conversationService := NewConversationService(conversationRepo)

	user, err := userService.Create(
		context.Background(),
		"renaldid",
		"Renaldi",
		"password123",
	)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	conversation, err := conversationService.Create(
		context.Background(),
		"General Chat",
	)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	memberService := NewConversationMemberService(
		conversationRepo,
		userRepo,
		memberRepo,
	)

	_, err = memberService.Add(
		context.Background(),
		conversation.ID,
		user.ID,
	)
	if err != nil {
		t.Fatalf("add member: %v", err)
	}

	svc := NewMessageService(
		messageRepo,
		conversationRepo,
		userRepo,
		memberRepo,
	)

	message, err := svc.Create(
		context.Background(),
		conversation.ID,
		user.ID,
		"Halo dari test",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if message.ID != 1 {
		t.Fatalf("expected message id 1, got %d", message.ID)
	}

	if message.Content != "Halo dari test" {
		t.Fatalf(
			"expected content Halo dari test, got %s",
			message.Content,
		)
	}
}

func TestMessageServiceCreateContentRequired(t *testing.T) {
	svc := NewMessageService(
		newMockMessageRepository(),
		newMockConversationRepository(),
		newMockUserRepository(),
		newMockConversationMemberRepository(),
	)

	_, err := svc.Create(
		context.Background(),
		1,
		1,
		"",
	)

	if !errors.Is(err, ErrMessageContentRequired) {
		t.Fatalf(
			"expected ErrMessageContentRequired, got %v",
			err,
		)
	}
}

func TestMessageServiceCreateConversationNotFound(t *testing.T) {
	svc := NewMessageService(
		newMockMessageRepository(),
		newMockConversationRepository(),
		newMockUserRepository(),
		newMockConversationMemberRepository(),
	)

	_, err := svc.Create(
		context.Background(),
		999,
		1,
		"Hello",
	)

	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf(
			"expected ErrConversationNotFound, got %v",
			err,
		)
	}
}

func TestMessageServiceCreateUserNotFound(t *testing.T) {
	conversationRepo := newMockConversationRepository()
	conversationService := NewConversationService(conversationRepo)

	conversation, _ := conversationService.Create(
		context.Background(),
		"General Chat",
	)

	svc := NewMessageService(
		newMockMessageRepository(),
		conversationRepo,
		newMockUserRepository(),
		newMockConversationMemberRepository(),
	)

	_, err := svc.Create(
		context.Background(),
		conversation.ID,
		999,
		"Hello",
	)

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf(
			"expected ErrUserNotFound, got %v",
			err,
		)
	}
}

func TestMessageServiceCreateSenderNotMember(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()

	userService := NewUserService(userRepo)
	conversationService := NewConversationService(conversationRepo)

	user, _ := userService.Create(
		context.Background(),
		"renaldid",
		"Renaldi",
		"password123",
	)

	conversation, _ := conversationService.Create(
		context.Background(),
		"General Chat",
	)

	svc := NewMessageService(
		newMockMessageRepository(),
		conversationRepo,
		userRepo,
		newMockConversationMemberRepository(),
	)

	_, err := svc.Create(
		context.Background(),
		conversation.ID,
		user.ID,
		"Hello",
	)

	if !errors.Is(err, ErrSenderNotConversationMember) {
		t.Fatalf(
			"expected ErrSenderNotConversationMember, got %v",
			err,
		)
	}
}
