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

func (m *mockConversationRepository) Create(
	ctx context.Context,
	conversation *model.Conversation,
) error {
	conversation.ID = int64(len(m.conversations) + 1)
	m.conversations[conversation.ID] = conversation

	return nil
}

func (m *mockConversationRepository) GetByID(
	ctx context.Context,
	id int64,
) (*model.Conversation, error) {
	conversation, ok := m.conversations[id]
	if !ok {
		return nil, nil
	}

	return conversation, nil
}

func TestConversationServiceCreate(t *testing.T) {
	repo := newMockConversationRepository()
	svc := NewConversationService(repo)

	conversation, err := svc.Create(
		context.Background(),
		"General Chat",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if conversation.ID != 1 {
		t.Fatalf("expected conversation id 1, got %d", conversation.ID)
	}

	if conversation.Title != "General Chat" {
		t.Fatalf(
			"expected title General Chat, got %s",
			conversation.Title,
		)
	}
}

func TestConversationServiceCreateTitleRequired(t *testing.T) {
	repo := newMockConversationRepository()
	svc := NewConversationService(repo)

	_, err := svc.Create(
		context.Background(),
		"",
	)

	if !errors.Is(err, ErrConversationTitleRequired) {
		t.Fatalf(
			"expected ErrConversationTitleRequired, got %v",
			err,
		)
	}
}

func TestConversationServiceGetByID(t *testing.T) {
	repo := newMockConversationRepository()
	svc := NewConversationService(repo)

	created, err := svc.Create(
		context.Background(),
		"General Chat",
	)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	conversation, err := svc.GetByID(
		context.Background(),
		created.ID,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if conversation.ID != created.ID {
		t.Fatalf(
			"expected conversation id %d, got %d",
			created.ID,
			conversation.ID,
		)
	}
}

func TestConversationServiceGetByIDNotFound(t *testing.T) {
	repo := newMockConversationRepository()
	svc := NewConversationService(repo)

	_, err := svc.GetByID(
		context.Background(),
		999,
	)

	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf(
			"expected ErrConversationNotFound, got %v",
			err,
		)
	}
}
