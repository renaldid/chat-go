package service

import (
	"context"
	"errors"
	"testing"

	"github.com/renaldid/chat-go/internal/model"
)

type mockConversationMemberRepository struct {
	members map[[2]int64]*model.ConversationMember
}

func newMockConversationMemberRepository() *mockConversationMemberRepository {
	return &mockConversationMemberRepository{
		members: make(map[[2]int64]*model.ConversationMember),
	}
}

func (m *mockConversationMemberRepository) Add(
	ctx context.Context,
	member *model.ConversationMember,
) error {
	key := [2]int64{
		member.ConversationID,
		member.UserID,
	}

	m.members[key] = member

	return nil
}

func (m *mockConversationMemberRepository) Exists(
	ctx context.Context,
	conversationID int64,
	userID int64,
) (bool, error) {
	key := [2]int64{
		conversationID,
		userID,
	}

	_, exists := m.members[key]

	return exists, nil
}

func TestConversationMemberServiceAdd(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()

	userService := NewUserService(userRepo)
	conversationService := NewConversationService(conversationRepo)

	user, err := userService.Create(
		context.Background(),
		"renaldid",
		"Renaldi",
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

	svc := NewConversationMemberService(
		conversationRepo,
		userRepo,
		memberRepo,
	)

	member, err := svc.Add(
		context.Background(),
		conversation.ID,
		user.ID,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if member.ConversationID != conversation.ID {
		t.Fatalf(
			"expected conversation id %d, got %d",
			conversation.ID,
			member.ConversationID,
		)
	}

	if member.UserID != user.ID {
		t.Fatalf(
			"expected user id %d, got %d",
			user.ID,
			member.UserID,
		)
	}
}

func TestConversationMemberServiceAddDuplicate(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()

	userService := NewUserService(userRepo)
	conversationService := NewConversationService(conversationRepo)

	user, _ := userService.Create(
		context.Background(),
		"renaldid",
		"Renaldi",
	)

	conversation, _ := conversationService.Create(
		context.Background(),
		"General Chat",
	)

	svc := NewConversationMemberService(
		conversationRepo,
		userRepo,
		memberRepo,
	)

	_, err := svc.Add(
		context.Background(),
		conversation.ID,
		user.ID,
	)
	if err != nil {
		t.Fatalf("expected first add to succeed, got %v", err)
	}

	_, err = svc.Add(
		context.Background(),
		conversation.ID,
		user.ID,
	)

	if !errors.Is(err, ErrConversationMemberExists) {
		t.Fatalf(
			"expected ErrConversationMemberExists, got %v",
			err,
		)
	}
}

func TestConversationMemberServiceConversationNotFound(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()

	svc := NewConversationMemberService(
		conversationRepo,
		userRepo,
		memberRepo,
	)

	_, err := svc.Add(
		context.Background(),
		999,
		1,
	)

	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf(
			"expected ErrConversationNotFound, got %v",
			err,
		)
	}
}

func TestConversationMemberServiceUserNotFound(t *testing.T) {
	userRepo := newMockUserRepository()
	conversationRepo := newMockConversationRepository()
	memberRepo := newMockConversationMemberRepository()

	conversationService := NewConversationService(conversationRepo)

	conversation, _ := conversationService.Create(
		context.Background(),
		"General Chat",
	)

	svc := NewConversationMemberService(
		conversationRepo,
		userRepo,
		memberRepo,
	)

	_, err := svc.Add(
		context.Background(),
		conversation.ID,
		999,
	)

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf(
			"expected ErrUserNotFound, got %v",
			err,
		)
	}
}
