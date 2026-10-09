package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renaldid/chat-go/internal/model"
)

type UserRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewUserRepositoryPostgres(db *pgxpool.Pool) *UserRepositoryPostgres {
	return &UserRepositoryPostgres{
		db: db,
	}
}

func (r *UserRepositoryPostgres) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (
			username,
			display_name
		)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		user.Username,
		user.DisplayName,
	).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *UserRepositoryPostgres) GetByID(
	ctx context.Context,
	id int64,
) (*model.User, error) {
	query := `
		SELECT
			id,
			username,
			display_name,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`

	user := &model.User{}

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (r *UserRepositoryPostgres) GetByUsername(
	ctx context.Context,
	username string,
) (*model.User, error) {
	query := `
		SELECT
			id,
			username,
			display_name,
			created_at,
			updated_at
		FROM users
		WHERE username = $1
	`

	user := &model.User{}

	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get user by username: %w", err)
	}

	return user, nil
}
