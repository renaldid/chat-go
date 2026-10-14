package service

import (
	"context"
	"testing"

	"github.com/renaldid/chat-go/internal/model"
)

type mockGroupConversationRepository struct{}

func (m *mockGroupConversationRepository) Create(
	ctx context.Context,
	conversation *model.Conversation,
	creatorID int64,
	memberIDs []int64,
) error {
	conversation.ID = 1
	conversation.Type = model.ConversationTypeGroup
	return nil
}

func (m *mockGroupConversationRepository) Rename(
	ctx context.Context,
	conversationID int64,
	title string,
) (*model.Conversation, error) {
	return &model.Conversation{
		ID:    conversationID,
		Type:  model.ConversationTypeGroup,
		Title: title,
	}, nil
}

func (m *mockGroupConversationRepository) Leave(
	ctx context.Context,
	conversationID int64,
	userID int64,
) error {
	return nil
}

type mockGroupMemberRepository struct {
	members map[[2]int64]*model.ConversationMember
}

func newMockGroupMemberRepository() *mockGroupMemberRepository {
	return &mockGroupMemberRepository{
		members: make(map[[2]int64]*model.ConversationMember),
	}
}

func (m *mockGroupMemberRepository) Get(
	ctx context.Context,
	conversationID int64,
	userID int64,
) (*model.ConversationMember, error) {
	return m.members[[2]int64{conversationID, userID}], nil
}

func (m *mockGroupMemberRepository) List(
	ctx context.Context,
	conversationID int64,
) ([]model.ConversationMemberView, error) {
	return []model.ConversationMemberView{}, nil
}

func (m *mockGroupMemberRepository) Add(
	ctx context.Context,
	member *model.ConversationMember,
) error {
	m.members[[2]int64{
		member.ConversationID,
		member.UserID,
	}] = member

	return nil
}

func (m *mockGroupMemberRepository) Remove(
	ctx context.Context,
	conversationID int64,
	userID int64,
) error {
	delete(
		m.members,
		[2]int64{conversationID, userID},
	)

	return nil
}

func (m *mockGroupMemberRepository) UpdateRole(
	ctx context.Context,
	conversationID int64,
	userID int64,
	role string,
) error {
	member := m.members[[2]int64{
		conversationID,
		userID,
	}]

	member.Role = role

	return nil
}

func TestGroupConversationServiceCreate(t *testing.T) {
	userRepo := newMockUserRepository()
	groupRepo := &mockGroupConversationRepository{}
	memberRepo := newMockGroupMemberRepository()
	conversationRepo := newMockConversationRepository()

	userService := NewUserService(userRepo)

	creator, _ := userService.Create(
		context.Background(),
		"owner",
		"Owner",
		"password123",
	)

	member, _ := userService.Create(
		context.Background(),
		"member",
		"Member",
		"password123",
	)

	svc := NewGroupConversationService(
		groupRepo,
		memberRepo,
		conversationRepo,
		userRepo,
	)

	conversation, err := svc.Create(
		context.Background(),
		creator.ID,
		"My Group",
		[]int64{member.ID},
	)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	if conversation.Type != model.ConversationTypeGroup {
		t.Fatalf(
			"expected group type, got %s",
			conversation.Type,
		)
	}
}

func TestGroupMemberCannotAddMember(t *testing.T) {
	userRepo := newMockUserRepository()
	groupRepo := &mockGroupConversationRepository{}
	memberRepo := newMockGroupMemberRepository()
	conversationRepo := newMockConversationRepository()

	conversation := &model.Conversation{
		Type:  model.ConversationTypeGroup,
		Title: "Group",
	}

	if err := conversationRepo.Create(
		context.Background(),
		conversation,
	); err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	memberRepo.members[[2]int64{
		conversation.ID,
		1,
	}] = &model.ConversationMember{
		ConversationID: conversation.ID,
		UserID:         1,
		Role:           model.ConversationRoleMember,
	}

	svc := NewGroupConversationService(
		groupRepo,
		memberRepo,
		conversationRepo,
		userRepo,
	)

	_, err := svc.AddMember(
		context.Background(),
		1,
		conversation.ID,
		2,
	)

	if err != ErrGroupPermissionDenied {
		t.Fatalf(
			"expected ErrGroupPermissionDenied, got %v",
			err,
		)
	}
}
