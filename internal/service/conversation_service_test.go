package service

import (
	"context"
	"errors"
	"testing"

	"github.com/renaldid/chat-go/internal/model"
)

type mockConversationRepository struct {
	conversations map[int64]*model.Conversation
}

func newMockConversationRepository() *mockConversationRepository {
	return &mockConversationRepository{
		conversations: make(map[int64]*model.Conversation),
	}
}

func (m *mockConversationRepository) Create(ctx context.Context, conversation *model.Conversation) error {
	conversation.ID = int64(len(m.conversations) + 1)
	m.conversations[conversation.ID] = conversation
	return nil
}

func (m *mockConversationRepository) GetByID(ctx context.Context, id int64) (*model.Conversation, error) {
	conversation, ok := m.conversations[id]
	if !ok {
		return nil, nil
	}
	return conversation, nil
}

func (m *mockConversationRepository) ListByUserID(ctx context.Context, userID int64, limit int, offset int) ([]model.ConversationSummary, error) {
	result := make([]model.ConversationSummary, 0)

	for _, conversation := range m.conversations {
		result = append(result, model.ConversationSummary{
			ID:           conversation.ID,
			Type:         conversation.Type,
			Title:        conversation.Title,
			DisplayTitle: conversation.Title,
			CreatedAt:    conversation.CreatedAt,
			UpdatedAt:    conversation.UpdatedAt,
		})
	}

	if offset >= len(result) {
		return []model.ConversationSummary{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

func TestConversationServiceCreate(t *testing.T) {
	repo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	svc := NewConversationService(repo, memberRepo)

	conversation, err := svc.Create(context.Background(), "General Chat", 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if conversation.ID != 1 {
		t.Fatalf("expected conversation id 1, got %d", conversation.ID)
	}

	exists, _ := memberRepo.Exists(context.Background(), conversation.ID, 1)
	if !exists {
		t.Fatal("expected creator to become conversation member")
	}
}

func TestConversationServiceCreateTitleRequired(t *testing.T) {
	svc := NewConversationService(
		newMockConversationRepository(),
		newMockConversationMemberRepository(),
	)

	_, err := svc.Create(context.Background(), "", 1)

	if !errors.Is(err, ErrConversationTitleRequired) {
		t.Fatalf("expected ErrConversationTitleRequired, got %v", err)
	}
}

func TestConversationServiceGetByID(t *testing.T) {
	repo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	svc := NewConversationService(repo, memberRepo)

	created, err := svc.Create(context.Background(), "General Chat", 1)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	conversation, err := svc.GetByID(context.Background(), created.ID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if conversation.ID != created.ID {
		t.Fatalf("expected conversation id %d, got %d", created.ID, conversation.ID)
	}
}

func TestConversationServiceGetByIDAccessDenied(t *testing.T) {
	repo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()
	svc := NewConversationService(repo, memberRepo)

	created, err := svc.Create(context.Background(), "General Chat", 1)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	_, err = svc.GetByID(context.Background(), created.ID, 2)

	if !errors.Is(err, ErrConversationAccessDenied) {
		t.Fatalf("expected ErrConversationAccessDenied, got %v", err)
	}
}

func TestConversationServiceGetByIDNotFound(t *testing.T) {
	svc := NewConversationService(
		newMockConversationRepository(),
		newMockConversationMemberRepository(),
	)

	_, err := svc.GetByID(context.Background(), 999, 1)

	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
}
