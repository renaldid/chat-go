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
	beforeID *int64,
	limit int,
) ([]model.Message, error) {
	result := make([]model.Message, 0)

	for _, message := range m.messages {
		if message.ConversationID != conversationID {
			continue
		}

		if beforeID != nil && message.ID >= *beforeID {
			continue
		}

		result = append(result, message)
	}

	if len(result) > limit {
		result = result[len(result)-limit:]
	}

	return result, nil
}

func (m *mockMessageRepository) GetByID(
	ctx context.Context,
	id int64,
) (*model.Message, error) {
	for i := range m.messages {
		if m.messages[i].ID == id {
			return &m.messages[i], nil
		}
	}

	return nil, nil
}

func TestMessageServiceCreate(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	messageRepo := newMockMessageRepository()

	userService := NewUserService(userRepo)
	conversationService := NewConversationService(conversationRepo, memberRepo)

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
		1,
	)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
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
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()

	conversationService := NewConversationService(
		conversationRepo,
		memberRepo,
	)

	conversation, err := conversationService.Create(
		context.Background(),
		"General Chat",
		1,
	)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	svc := NewMessageService(
		newMockMessageRepository(),
		conversationRepo,
		userRepo,
		memberRepo,
	)

	_, err = svc.Create(
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
	memberRepo := newMockConversationMemberRepository()

	userService := NewUserService(userRepo)
	conversationService := NewConversationService(
		conversationRepo,
		memberRepo,
	)

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
		999,
	)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	svc := NewMessageService(
		newMockMessageRepository(),
		conversationRepo,
		userRepo,
		memberRepo,
	)

	_, err = svc.Create(
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

func TestMessageServiceGetByConversationIDPagination(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	messageRepo := newMockMessageRepository()

	userService := NewUserService(userRepo)
	conversationService := NewConversationService(
		conversationRepo,
		memberRepo,
	)

	user, err := userService.Create(
		context.Background(),
		"pagination-user",
		"Pagination User",
		"password123",
	)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	conversation, err := conversationService.Create(
		context.Background(),
		"Pagination Chat",
		user.ID,
	)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	svc := NewMessageService(
		messageRepo,
		conversationRepo,
		userRepo,
		memberRepo,
	)

	for _, content := range []string{"message 1", "message 2", "message 3"} {
		_, err := svc.Create(
			context.Background(),
			conversation.ID,
			user.ID,
			content,
		)
		if err != nil {
			t.Fatalf("create message: %v", err)
		}
	}

	page1, err := svc.GetByConversationID(
		context.Background(),
		conversation.ID,
		user.ID,
		nil,
		2,
	)
	if err != nil {
		t.Fatalf("get first page: %v", err)
	}

	if len(page1.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(page1.Messages))
	}

	if page1.Messages[0].ID != 2 || page1.Messages[1].ID != 3 {
		t.Fatalf("expected message ids 2 and 3")
	}

	if !page1.HasMore {
		t.Fatal("expected first page to have more messages")
	}

	if page1.NextCursor == nil || *page1.NextCursor != 2 {
		t.Fatal("expected next cursor 2")
	}

	page2, err := svc.GetByConversationID(
		context.Background(),
		conversation.ID,
		user.ID,
		page1.NextCursor,
		2,
	)
	if err != nil {
		t.Fatalf("get second page: %v", err)
	}

	if len(page2.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(page2.Messages))
	}

	if page2.Messages[0].ID != 1 {
		t.Fatalf("expected message id 1, got %d", page2.Messages[0].ID)
	}

	if page2.HasMore {
		t.Fatal("expected second page to have no more messages")
	}

	if page2.NextCursor != nil {
		t.Fatal("expected next cursor to be nil")
	}
}
