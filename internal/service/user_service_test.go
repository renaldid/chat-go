package service

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/renaldid/chat-go/internal/model"
)

type mockUserRepository struct {
	users map[int64]*model.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[int64]*model.User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *model.User) error {
	user.ID = int64(len(m.users) + 1)
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, nil
	}

	return user, nil
}

func (m *mockUserRepository) GetByUsername(
	ctx context.Context,
	username string,
) (*model.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}

	return nil, nil
}

func TestUserServiceCreate(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewUserService(repo)

	password := "password123"

	user, err := svc.Create(
		context.Background(),
		"renaldid",
		"Renaldi",
		password,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID != 1 {
		t.Fatalf("expected user id 1, got %d", user.ID)
	}

	if user.Username != "renaldid" {
		t.Fatalf("expected username renaldid, got %s", user.Username)
	}

	if user.PasswordHash == "" {
		t.Fatal("expected password hash to be set")
	}

	if user.PasswordHash == password {
		t.Fatal("password must not be stored as plain text")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	); err != nil {
		t.Fatalf("password hash does not match password: %v", err)
	}
}

func TestUserServiceCreateUsernameRequired(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"",
		"Renaldi",
		"password123",
	)

	if !errors.Is(err, ErrUsernameRequired) {
		t.Fatalf("expected ErrUsernameRequired, got %v", err)
	}
}

func TestUserServiceCreateDisplayNameRequired(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"renaldid",
		"",
		"password123",
	)

	if !errors.Is(err, ErrDisplayNameRequired) {
		t.Fatalf("expected ErrDisplayNameRequired, got %v", err)
	}
}

func TestUserServiceCreatePasswordRequired(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"renaldid",
		"Renaldi",
		"",
	)

	if !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("expected ErrPasswordRequired, got %v", err)
	}
}

func TestUserServiceCreatePasswordTooShort(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"renaldid",
		"Renaldi",
		"1234567",
	)

	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestUserServiceCreateDuplicateUsername(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"renaldid",
		"Renaldi",
		"password123",
	)
	if err != nil {
		t.Fatalf("expected first create to succeed, got %v", err)
	}

	_, err = svc.Create(
		context.Background(),
		"renaldid",
		"Another User",
		"password456",
	)

	if !errors.Is(err, ErrUsernameExists) {
		t.Fatalf("expected ErrUsernameExists, got %v", err)
	}
}

func TestUserServiceGetByID(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewUserService(repo)

	created, err := svc.Create(
		context.Background(),
		"renaldid",
		"Renaldi",
		"password123",
	)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	user, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID != created.ID {
		t.Fatalf("expected user id %d, got %d", created.ID, user.ID)
	}
}

func TestUserServiceGetByIDNotFound(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewUserService(repo)

	_, err := svc.GetByID(context.Background(), 999)

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
