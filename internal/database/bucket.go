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

func (s *service) DeleteBucket(ctx context.Context, name string, ownerId int, region string) error {
	query := `DELETE FROM buckets WHERE name = $1 AND owner_id = $2 AND region = $3`
	_, err := s.db.Exec(ctx, query, name, ownerId, region)
	return err
}

func (s *service) ListBucketsByOwnerId(ctx context.Context, ownerId int) ([]Bucket, error) {
	query := `
		SELECT id, name, owner_id, region, created_at
		FROM buckets
		WHERE owner_id = $1
		ORDER BY name
	`

	rows, err := s.db.Query(ctx, query, ownerId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []Bucket
	for rows.Next() {
		var bucket Bucket
		err := rows.Scan(
			&bucket.Id,
			&bucket.Name,
			&bucket.OwnerId,
			&bucket.Region,
			&bucket.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		buckets = append(buckets, bucket)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return buckets, nil
}

func (s *service) IsBucketEmpty(ctx context.Context, bucketId int, bucketName string) (bool, error) {
	var hasObjects bool
	objQuery := `SELECT EXISTS(SELECT 1 FROM objects WHERE bucket_id = $1 LIMIT 1)`
	if err := s.db.QueryRow(ctx, objQuery, bucketId).Scan(&hasObjects); err != nil {
		return false, err
	}
	if hasObjects {
		return false, nil
	}

	// 2. Check active multipart uploads
	var hasUploads bool
	uploadQuery := `SELECT EXISTS(SELECT 1 FROM multipart_uploads WHERE bucket_name = $1 LIMIT 1)`
	if err := s.db.QueryRow(ctx, uploadQuery, bucketName).Scan(&hasUploads); err != nil {
		return false, err
	}
	if hasUploads {
		return false, nil
	}

	return true, nil
}
