package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Bucket struct {
	Id        int
	Name      string
	OwnerId   int
	Region    string
	CreatedAt time.Time
}

func (s *service) CreateBucket(ctx context.Context, name string, ownerId int, region string) (*Bucket, error) {
	var bucket Bucket

	query := `
       INSERT INTO buckets (name, owner_id, region)
       VALUES ($1, $2, $3)
       RETURNING id, name, owner_id, region, created_at
    `

	err := s.db.QueryRow(ctx, query, name, ownerId, region).Scan(
		&bucket.Id,
		&bucket.Name,
		&bucket.OwnerId,
		&bucket.Region,
		&bucket.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &bucket, nil
}

func (s *service) GetBucketByName(ctx context.Context, name string) (*Bucket, error) {
	var bucket Bucket

	query := `
		SELECT id, name, owner_id, region, created_at
		FROM buckets
		WHERE name = $1
	`

	err := s.db.QueryRow(ctx, query, name).Scan(
		&bucket.Id,
		&bucket.Name,
		&bucket.OwnerId,
		&bucket.Region,
		&bucket.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &bucket, nil
}
