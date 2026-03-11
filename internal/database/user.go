package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type RegisterParams struct {
	Name  string
	Email string
}

type User struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (s *service) CreateUser(ctx context.Context, params RegisterParams) (*User, error) {
	var user User

	query := `
		INSERT INTO users (name, email)
		VALUES ($1, $2)
		RETURNING id, name, email
	`

	err := s.db.QueryRow(ctx, query, params.Name, params.Email).Scan(
		&user.Id,
		&user.Name,
		&user.Email,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *service) GetUser(ctx context.Context, id int) (*User, error) {
	var user User

	query := `
		SELECT id, name, email
		FROM users
		WHERE id = $1
	`

	err := s.db.QueryRow(ctx, query, id).Scan(
		&user.Id,
		&user.Name,
		&user.Email,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (s *service) GetUserByName(ctx context.Context, name string) (*User, error) {
	var user User

	query := `
		SELECT id, name, email
		FROM users
		WHERE name = $1
	`

	err := s.db.QueryRow(ctx, query, name).Scan(
		&user.Id,
		&user.Name,
		&user.Email,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User

	query := `
		SELECT id, name, email
		FROM users
		WHERE email = $1
	`

	err := s.db.QueryRow(ctx, query, email).Scan(
		&user.Id,
		&user.Name,
		&user.Email,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
