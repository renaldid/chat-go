package service

import (
	"context"
	"errors"
	"testing"

	"github.com/renaldid/chat-go/internal/model"
)

type mockDirectConversationRepository struct {
	conversation *model.Conversation
	created      bool
	err          error
	userID       int64
	targetUserID int64
}

func (
	m *mockDirectConversationRepository,
) CreateOrGet(
	ctx context.Context,
	userID int64,
	targetUserID int64,
) (*model.Conversation, bool, error) {
	m.userID = userID
	m.targetUserID = targetUserID

	if m.err != nil {
		return nil, false, m.err
	}

	return m.conversation, m.created, nil
}

func TestDirectConversationServiceCreate(t *testing.T) {
	userRepo := newMockUserRepository()

	userService := NewUserService(userRepo)

	user1, err := userService.Create(
		context.Background(),
		"user1",
		"User One",
		"password123",
	)
	if err != nil {
		t.Fatalf("create user 1: %v", err)
	}

	user2, err := userService.Create(
		context.Background(),
		"user2",
		"User Two",
		"password123",
	)
	if err != nil {
		t.Fatalf("create user 2: %v", err)
	}

	directRepo := &mockDirectConversationRepository{
		conversation: &model.Conversation{
			ID: 10,
		},
		created: true,
	}

	svc := NewDirectConversationService(
		directRepo,
		userRepo,
	)

	conversation, created, err := svc.CreateOrGet(
		context.Background(),
		user1.ID,
		user2.ID,
	)
	if err != nil {
		t.Fatalf("create direct conversation: %v", err)
	}

	if !created {
		t.Fatal("expected conversation to be created")
	}

	if conversation.ID != 10 {
		t.Fatalf(
			"expected conversation id 10, got %d",
			conversation.ID,
		)
	}

	if directRepo.userID != user1.ID {
		t.Fatalf(
			"expected user id %d, got %d",
			user1.ID,
			directRepo.userID,
		)
	}

	if directRepo.targetUserID != user2.ID {
		t.Fatalf(
			"expected target user id %d, got %d",
			user2.ID,
			directRepo.targetUserID,
		)
	}
}

func TestDirectConversationServiceExisting(
	t *testing.T,
) {
	userRepo := newMockUserRepository()

	userService := NewUserService(userRepo)

	user1, _ := userService.Create(
		context.Background(),
		"user1",
		"User One",
		"password123",
	)

	user2, _ := userService.Create(
		context.Background(),
		"user2",
		"User Two",
		"password123",
	)

	directRepo := &mockDirectConversationRepository{
		conversation: &model.Conversation{
			ID: 10,
		},
		created: false,
	}

	svc := NewDirectConversationService(
		directRepo,
		userRepo,
	)

	conversation, created, err := svc.CreateOrGet(
		context.Background(),
		user1.ID,
		user2.ID,
	)
	if err != nil {
		t.Fatalf("get direct conversation: %v", err)
	}

	if created {
		t.Fatal("expected existing conversation")
	}

	if conversation.ID != 10 {
		t.Fatalf(
			"expected conversation id 10, got %d",
			conversation.ID,
		)
	}
}

func TestDirectConversationServiceSelf(
	t *testing.T,
) {
	userRepo := newMockUserRepository()

	directRepo :=
		&mockDirectConversationRepository{}

	svc := NewDirectConversationService(
		directRepo,
		userRepo,
	)

	_, _, err := svc.CreateOrGet(
		context.Background(),
		1,
		1,
	)

	if !errors.Is(
		err,
		ErrDirectConversationWithSelf,
	) {
		t.Fatalf(
			"expected ErrDirectConversationWithSelf, got %v",
			err,
		)
	}
}

func TestDirectConversationServiceUserNotFound(
	t *testing.T,
) {
	userRepo := newMockUserRepository()

	directRepo :=
		&mockDirectConversationRepository{}

	svc := NewDirectConversationService(
		directRepo,
		userRepo,
	)

	_, _, err := svc.CreateOrGet(
		context.Background(),
		1,
		999,
	)

	if !errors.Is(
		err,
		ErrUserNotFound,
	) {
		t.Fatalf(
			"expected ErrUserNotFound, got %v",
			err,
		)
	}
}
