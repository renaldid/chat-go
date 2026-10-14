package service

import (
	"context"
	"errors"
	"testing"
	"time"

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

func (m *mockMessageRepository) UpdateContent(ctx context.Context, id int64, content string) (*model.Message, error) {
	for i := range m.messages {
		if m.messages[i].ID == id {
			if m.messages[i].DeletedAt != nil {
				return nil, nil
			}

			now := time.Now()
			m.messages[i].Content = content
			m.messages[i].EditedAt = &now
			return &m.messages[i], nil
		}
	}

	return nil, nil
}

func (m *mockMessageRepository) SoftDelete(ctx context.Context, id int64) (*model.Message, error) {
	for i := range m.messages {
		if m.messages[i].ID == id {
			if m.messages[i].DeletedAt != nil {
				return nil, nil
			}

			now := time.Now()
			m.messages[i].Content = ""
			m.messages[i].DeletedAt = &now
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

func TestMessageServiceUpdate(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	messageRepo := newMockMessageRepository()

	userService := NewUserService(userRepo)
	user, err := userService.Create(context.Background(), "editor", "Editor", "password123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	conversationService := NewConversationService(conversationRepo, memberRepo)
	conversation, err := conversationService.Create(context.Background(), "Edit Test", user.ID)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	svc := NewMessageService(messageRepo, conversationRepo, userRepo, memberRepo)

	message, err := svc.Create(context.Background(), conversation.ID, user.ID, "before edit")
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	updated, err := svc.Update(context.Background(), conversation.ID, user.ID, message.ID, "after edit")
	if err != nil {
		t.Fatalf("update message: %v", err)
	}

	if updated.Content != "after edit" {
		t.Fatalf("expected after edit, got %s", updated.Content)
	}

	if updated.EditedAt == nil {
		t.Fatal("expected edited_at")
	}
}

func TestMessageServiceUpdatePermissionDenied(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	messageRepo := newMockMessageRepository()

	userService := NewUserService(userRepo)
	user1, _ := userService.Create(context.Background(), "sender", "Sender", "password123")
	user2, _ := userService.Create(context.Background(), "other", "Other", "password123")

	conversationService := NewConversationService(conversationRepo, memberRepo)
	conversation, _ := conversationService.Create(context.Background(), "Edit Test", user1.ID)

	err := memberRepo.Add(context.Background(), &model.ConversationMember{
		ConversationID: conversation.ID,
		UserID:         user2.ID,
		Role:           model.ConversationRoleMember,
	})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}

	svc := NewMessageService(messageRepo, conversationRepo, userRepo, memberRepo)
	message, _ := svc.Create(context.Background(), conversation.ID, user1.ID, "hello")

	_, err = svc.Update(context.Background(), conversation.ID, user2.ID, message.ID, "hacked")
	if !errors.Is(err, ErrMessagePermissionDenied) {
		t.Fatalf("expected ErrMessagePermissionDenied, got %v", err)
	}
}

func TestMessageServiceDelete(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	messageRepo := newMockMessageRepository()

	userService := NewUserService(userRepo)
	user, _ := userService.Create(context.Background(), "deleter", "Deleter", "password123")

	conversationService := NewConversationService(conversationRepo, memberRepo)
	conversation, _ := conversationService.Create(context.Background(), "Delete Test", user.ID)

	svc := NewMessageService(messageRepo, conversationRepo, userRepo, memberRepo)
	message, _ := svc.Create(context.Background(), conversation.ID, user.ID, "delete me")

	deleted, changed, err := svc.Delete(context.Background(), conversation.ID, user.ID, message.ID)
	if err != nil {
		t.Fatalf("delete message: %v", err)
	}
	if !changed {
		t.Fatal("expected first delete to change message")
	}
	if deleted.DeletedAt == nil {
		t.Fatal("expected deleted_at")
	}
	if deleted.Content != "" {
		t.Fatal("expected deleted message content to be empty")
	}

	_, changed, err = svc.Delete(context.Background(), conversation.ID, user.ID, message.ID)
	if err != nil {
		t.Fatalf("delete second time: %v", err)
	}
	if changed {
		t.Fatal("expected second delete to be idempotent")
	}
}
