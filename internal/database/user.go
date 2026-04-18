package database

import (
	"context"
	"errors"
	"time"

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

type ApiKey struct {
	Id              int       `json:"id"`
	UserId          int       `json:"user_id"`
	AccessKey       string    `json:"access_key"`
	SecretKey       string    `json:"secret_key"`
	PermissionLevel string    `json:"permission_level"`
	CreatedAt       time.Time `json:"created_at"`
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

func (s *service) GetUserByApiKey(ctx context.Context, accessKey string) (*User, *ApiKey, error) {
	var user User
	var apiKey ApiKey

	query := `
		SELECT 
			u.id, u.name, u.email, 
			ak.id, ak.user_id, ak.access_key, ak.secret_key, ak.permission_level, ak.created_at
		FROM users u
		JOIN api_keys ak ON u.id = ak.user_id
		WHERE ak.access_key = $1
	`

	err := s.db.QueryRow(ctx, query, accessKey).Scan(
		&user.Id, &user.Name, &user.Email,
		&apiKey.Id, &apiKey.UserId, &apiKey.AccessKey, &apiKey.SecretKey, &apiKey.PermissionLevel, &apiKey.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	return &user, &apiKey, nil
}

func (s *service) CreateApiKey(ctx context.Context, userId int, permissionLevel string) (*ApiKey, error) {
	var apiKey ApiKey

	query := `
		INSERT INTO api_keys (user_id, access_key, secret_key, permission_level)
		VALUES ($1, gen_random_uuid()::text, gen_random_uuid()::text, $2)
		RETURNING id, user_id, access_key, secret_key, permission_level, created_at
	`

	err := s.db.QueryRow(ctx, query, userId, permissionLevel).Scan(
		&apiKey.Id,
		&apiKey.UserId,
		&apiKey.AccessKey,
		&apiKey.SecretKey,
		&apiKey.PermissionLevel,
		&apiKey.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &apiKey, nil
}

func (s *service) GetApiKeysByUserId(ctx context.Context, userId int) ([]ApiKey, error) {
	var keys []ApiKey

	query := `
		SELECT id, user_id, access_key, secret_key, permission_level, created_at
		FROM api_keys
		WHERE user_id = $1
	`
	rows, err := s.db.Query(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var apiKey ApiKey
		err := rows.Scan(
			&apiKey.Id,
			&apiKey.UserId,
			&apiKey.AccessKey,
			&apiKey.SecretKey,
			&apiKey.PermissionLevel,
			&apiKey.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		keys = append(keys, apiKey)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}
