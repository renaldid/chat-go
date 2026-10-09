package service

import (
	"context"
	"errors"
	"testing"

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

	user, err := svc.Create(
		context.Background(),
		"renaldid",
		"Renaldi",
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
}

func TestUserServiceCreateUsernameRequired(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"",
		"Renaldi",
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
	)

	if !errors.Is(err, ErrDisplayNameRequired) {
		t.Fatalf("expected ErrDisplayNameRequired, got %v", err)
	}
}

func TestUserServiceCreateDuplicateUsername(t *testing.T) {
	repo := newMockUserRepository()
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"renaldid",
		"Renaldi",
	)
	if err != nil {
		t.Fatalf("expected first create to succeed, got %v", err)
	}

	_, err = svc.Create(
		context.Background(),
		"renaldid",
		"Another User",
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
