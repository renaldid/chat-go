package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/renaldid/chat-go/internal/auth"
	"github.com/renaldid/chat-go/internal/repository"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

type AuthService struct {
	userRepository repository.UserRepository
	jwtManager     *auth.JWTManager
}

func NewAuthService(
	userRepository repository.UserRepository,
	jwtManager *auth.JWTManager,
) *AuthService {
	return &AuthService{
		userRepository: userRepository,
		jwtManager:     jwtManager,
	}
}

func (s *AuthService) Login(
	ctx context.Context,
	username string,
	password string,
) (string, error) {
	user, err := s.userRepository.GetByUsername(ctx, username)
	if err != nil {
		return "", err
	}

	if user == nil {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := s.jwtManager.Generate(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}
