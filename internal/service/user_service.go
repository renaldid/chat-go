package service

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/renaldid/chat-go/internal/model"
	"github.com/renaldid/chat-go/internal/repository"
)

var (
	ErrUsernameRequired    = errors.New("username is required")
	ErrDisplayNameRequired = errors.New("display name is required")
	ErrPasswordRequired    = errors.New("password is required")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters")
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
	password string,
) (*model.User, error) {
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)

	if username == "" {
		return nil, ErrUsernameRequired
	}

	if displayName == "" {
		return nil, ErrDisplayNameRequired
	}

	if password == "" {
		return nil, ErrPasswordRequired
	}

	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}

	existingUser, err := s.userRepository.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if existingUser != nil {
		return nil, ErrUsernameExists
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: string(passwordHash),
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
