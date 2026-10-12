package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/renaldid/chat-go/internal/auth"
	"github.com/renaldid/chat-go/internal/model"
)

func TestAuthServiceLogin(t *testing.T) {
	repo := newMockUserRepository()

	password := "password123"

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := &model.User{
		ID:           1,
		Username:     "authUser",
		DisplayName:  "Auth User",
		PasswordHash: string(passwordHash),
	}

	repo.users[user.ID] = user

	jwtManager := auth.NewJWTManager(
		"test-secret",
		time.Hour,
	)

	svc := NewAuthService(repo, jwtManager)

	token, err := svc.Login(
		context.Background(),
		"authUser",
		password,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token == "" {
		t.Fatal("expected token to be generated")
	}

	claims, err := jwtManager.Parse(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	if claims.UserID != user.ID {
		t.Fatalf(
			"expected user id %d, got %d",
			user.ID,
			claims.UserID,
		)
	}
}

func TestAuthServiceLoginUserNotFound(t *testing.T) {
	repo := newMockUserRepository()

	jwtManager := auth.NewJWTManager(
		"test-secret",
		time.Hour,
	)

	svc := NewAuthService(repo, jwtManager)

	_, err := svc.Login(
		context.Background(),
		"unknown",
		"password123",
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf(
			"expected ErrInvalidCredentials, got %v",
			err,
		)
	}
}

func TestAuthServiceLoginInvalidPassword(t *testing.T) {
	repo := newMockUserRepository()

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte("password123"),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	repo.users[1] = &model.User{
		ID:           1,
		Username:     "authUser",
		DisplayName:  "Auth User",
		PasswordHash: string(passwordHash),
	}

	jwtManager := auth.NewJWTManager(
		"test-secret",
		time.Hour,
	)

	svc := NewAuthService(repo, jwtManager)

	_, err = svc.Login(
		context.Background(),
		"authUser",
		"wrong-password",
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf(
			"expected ErrInvalidCredentials, got %v",
			err,
		)
	}
}
