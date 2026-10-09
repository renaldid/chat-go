package service

import (
	"context"
	"errors"
	"strings"

	"github.com/renaldid/chat-go/internal/model"
	"github.com/renaldid/chat-go/internal/repository"
)

var (
	ErrUsernameRequired    = errors.New("username is required")
	ErrDisplayNameRequired = errors.New("display name is required")
	ErrUsernameExists      = errors.New("username already exists")
	ErrUserNotFound        = errors.New("user not found")
)

type UserService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) *UserService {
	return &UserService{
		userRepository: userRepository,
	}
}

func (s *UserService) Create(
	ctx context.Context,
	username string,
	displayName string,
) (*model.User, error) {
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)

	if username == "" {
		return nil, ErrUsernameRequired
	}

	if displayName == "" {
		return nil, ErrDisplayNameRequired
	}

	existingUser, err := s.userRepository.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if existingUser != nil {
		return nil, ErrUsernameExists
	}

	user := &model.User{
		Username:    username,
		DisplayName: displayName,
	}

	if err := s.userRepository.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetByID(
	ctx context.Context,
	id int64,
) (*model.User, error) {
	user, err := s.userRepository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}
