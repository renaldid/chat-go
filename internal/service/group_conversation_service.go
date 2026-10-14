package service

import (
	"context"
	"errors"
	"strings"

	"github.com/renaldid/chat-go/internal/model"
	"github.com/renaldid/chat-go/internal/repository"
)

var (
	ErrGroupConversationRequired  = errors.New("conversation is not a group")
	ErrGroupPermissionDenied      = errors.New("insufficient group permission")
	ErrConversationMemberNotFound = errors.New("conversation member not found")
	ErrCannotRemoveGroupOwner     = errors.New("group owner cannot be removed")
	ErrCannotChangeGroupOwnerRole = errors.New("group owner role cannot be changed")
	ErrInvalidConversationRole    = errors.New("invalid conversation role")
	ErrUseLeaveConversation       = errors.New("use leave conversation to remove yourself")
)

type GroupConversationService struct {
	groupRepository        repository.GroupConversationRepository
	memberRepository       repository.GroupMemberRepository
	conversationRepository repository.ConversationRepository
	userRepository         repository.UserRepository
}

func NewGroupConversationService(
	groupRepository repository.GroupConversationRepository,
	memberRepository repository.GroupMemberRepository,
	conversationRepository repository.ConversationRepository,
	userRepository repository.UserRepository,
) *GroupConversationService {
	return &GroupConversationService{
		groupRepository:        groupRepository,
		memberRepository:       memberRepository,
		conversationRepository: conversationRepository,
		userRepository:         userRepository,
	}
}

func (s *GroupConversationService) Create(
	ctx context.Context,
	creatorID int64,
	title string,
	memberIDs []int64,
) (*model.Conversation, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrConversationTitleRequired
	}

	uniqueMemberIDs := make([]int64, 0, len(memberIDs))
	seen := map[int64]struct{}{
		creatorID: {},
	}

	for _, memberID := range memberIDs {
		if memberID <= 0 {
			return nil, ErrUserNotFound
		}

		if _, exists := seen[memberID]; exists {
			continue
		}

		user, err := s.userRepository.GetByID(ctx, memberID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, ErrUserNotFound
		}

		seen[memberID] = struct{}{}
		uniqueMemberIDs = append(uniqueMemberIDs, memberID)
	}

	conversation := &model.Conversation{
		Type:  model.ConversationTypeGroup,
		Title: title,
	}

	if err := s.groupRepository.Create(
		ctx,
		conversation,
		creatorID,
		uniqueMemberIDs,
	); err != nil {
		return nil, err
	}

	return conversation, nil
}

func (s *GroupConversationService) Rename(
	ctx context.Context,
	requesterID int64,
	conversationID int64,
	title string,
) (*model.Conversation, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrConversationTitleRequired
	}

	member, err := s.requireGroupMember(ctx, conversationID, requesterID)
	if err != nil {
		return nil, err
	}

	if member.Role != model.ConversationRoleOwner &&
		member.Role != model.ConversationRoleAdmin {
		return nil, ErrGroupPermissionDenied
	}

	return s.groupRepository.Rename(ctx, conversationID, title)
}

func (s *GroupConversationService) ListMembers(
	ctx context.Context,
	requesterID int64,
	conversationID int64,
) ([]model.ConversationMemberView, error) {
	if _, err := s.requireGroupMember(
		ctx,
		conversationID,
		requesterID,
	); err != nil {
		return nil, err
	}

	return s.memberRepository.List(ctx, conversationID)
}

func (s *GroupConversationService) AddMember(
	ctx context.Context,
	requesterID int64,
	conversationID int64,
	userID int64,
) (*model.ConversationMember, error) {
	requester, err := s.requireGroupMember(
		ctx,
		conversationID,
		requesterID,
	)
	if err != nil {
		return nil, err
	}

	if requester.Role != model.ConversationRoleOwner &&
		requester.Role != model.ConversationRoleAdmin {
		return nil, ErrGroupPermissionDenied
	}

	user, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	existing, err := s.memberRepository.Get(
		ctx,
		conversationID,
		userID,
	)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrConversationMemberExists
	}

	member := &model.ConversationMember{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           model.ConversationRoleMember,
	}

	if err := s.memberRepository.Add(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

func (s *GroupConversationService) RemoveMember(
	ctx context.Context,
	requesterID int64,
	conversationID int64,
	targetUserID int64,
) error {
	requester, err := s.requireGroupMember(
		ctx,
		conversationID,
		requesterID,
	)
	if err != nil {
		return err
	}

	if requesterID == targetUserID {
		return ErrUseLeaveConversation
	}

	target, err := s.memberRepository.Get(
		ctx,
		conversationID,
		targetUserID,
	)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrConversationMemberNotFound
	}

	if target.Role == model.ConversationRoleOwner {
		return ErrCannotRemoveGroupOwner
	}

	switch requester.Role {
	case model.ConversationRoleOwner:
	case model.ConversationRoleAdmin:
		if target.Role != model.ConversationRoleMember {
			return ErrGroupPermissionDenied
		}
	default:
		return ErrGroupPermissionDenied
	}

	return s.memberRepository.Remove(
		ctx,
		conversationID,
		targetUserID,
	)
}

func (s *GroupConversationService) UpdateRole(
	ctx context.Context,
	requesterID int64,
	conversationID int64,
	targetUserID int64,
	role string,
) error {
	requester, err := s.requireGroupMember(
		ctx,
		conversationID,
		requesterID,
	)
	if err != nil {
		return err
	}

	if requester.Role != model.ConversationRoleOwner {
		return ErrGroupPermissionDenied
	}

	if role != model.ConversationRoleAdmin &&
		role != model.ConversationRoleMember {
		return ErrInvalidConversationRole
	}

	target, err := s.memberRepository.Get(
		ctx,
		conversationID,
		targetUserID,
	)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrConversationMemberNotFound
	}

	if target.Role == model.ConversationRoleOwner {
		return ErrCannotChangeGroupOwnerRole
	}

	return s.memberRepository.UpdateRole(
		ctx,
		conversationID,
		targetUserID,
		role,
	)
}

func (s *GroupConversationService) Leave(
	ctx context.Context,
	userID int64,
	conversationID int64,
) error {
	if _, err := s.requireGroupMember(
		ctx,
		conversationID,
		userID,
	); err != nil {
		return err
	}

	return s.groupRepository.Leave(
		ctx,
		conversationID,
		userID,
	)
}

func (s *GroupConversationService) requireGroupMember(
	ctx context.Context,
	conversationID int64,
	userID int64,
) (*model.ConversationMember, error) {
	conversation, err := s.conversationRepository.GetByID(
		ctx,
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	if conversation.Type != model.ConversationTypeGroup {
		return nil, ErrGroupConversationRequired
	}

	member, err := s.memberRepository.Get(
		ctx,
		conversationID,
		userID,
	)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrConversationAccessDenied
	}

	return member, nil
}
