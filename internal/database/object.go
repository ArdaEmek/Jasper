package database

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Object struct {
	ObjectId           string
	BucketId           int
	ObjectKey          string
	SizeBytes          int64
	ContentType        string
	ContentDisposition string
	ContentLanguage    string
	CustomMetadata     map[string]string
	ETag               string
	CreatedAt          time.Time
}

type ListObjectsV2Result struct {
	Objects               []Object
	CommonPrefixes        []string
	NextContinuationToken string
	IsTruncated           bool
}

func (s *service) GetObject(ctx context.Context, objectId string) (*Object, error) {
	var obj Object
	var customMetaBytes []byte

	query := `
		SELECT object_id, bucket_id, object_key, size_bytes, content_type, etag, content_disposition, content_language, custom_metadata, created_at
		FROM objects
		WHERE object_id = $1
	`

	err := s.db.QueryRow(ctx, query, objectId).Scan(
		&obj.ObjectId,
		&obj.BucketId,
		&obj.ObjectKey,
		&obj.SizeBytes,
		&obj.ContentType,
		&obj.ETag,
		&obj.ContentDisposition,
		&obj.ContentLanguage,
		&customMetaBytes,
		&obj.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if len(customMetaBytes) > 0 {
		_ = json.Unmarshal(customMetaBytes, &obj.CustomMetadata)
	}

	return &obj, nil
}

func (s *service) GetObjectByKey(ctx context.Context, bucketId int, objectKey string) (*Object, error) {
	var obj Object
	var customMetaBytes []byte

	query := `
		SELECT object_id, bucket_id, object_key, size_bytes, content_type, etag, content_disposition, content_language, custom_metadata, created_at
		FROM objects
		WHERE bucket_id = $1 AND object_key = $2
	`

	err := s.db.QueryRow(ctx, query, bucketId, objectKey).Scan(
		&obj.ObjectId,
		&obj.BucketId,
		&obj.ObjectKey,
		&obj.SizeBytes,
		&obj.ContentType,
		&obj.ETag,
		&obj.ContentDisposition,
		&obj.ContentLanguage,
		&customMetaBytes,
		&obj.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if len(customMetaBytes) > 0 {
		_ = json.Unmarshal(customMetaBytes, &obj.CustomMetadata)
	}

	return &obj, nil
}

func (s *service) ListObjectsV2(
	ctx context.Context,
	bucketId int,
	prefix string,
	continuationToken string,
	startAfter string,
	maxKeys int,
) (*ListObjectsV2Result, error) {
	marker := startAfter
	if continuationToken != "" {
		marker = continuationToken
	}

	// Fetch maxKeys + 1 items to check for truncation
	query := `
		SELECT object_id, bucket_id, object_key, size_bytes, content_type, etag, 
		       content_disposition, content_language, created_at
		FROM objects
		WHERE bucket_id = $1
		  AND ($2 = '' OR object_key LIKE $2 || '%')
		  AND ($3 = '' OR object_key > $3)
		ORDER BY object_key ASC
		LIMIT $4
	`

	rows, err := s.db.Query(ctx, query, bucketId, prefix, marker, maxKeys+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []Object
	for rows.Next() {
		var obj Object
		err := rows.Scan(
			&obj.ObjectId,
			&obj.BucketId,
			&obj.ObjectKey,
			&obj.SizeBytes,
			&obj.ContentType,
			&obj.ETag,
			&obj.ContentDisposition,
			&obj.ContentLanguage,
			&obj.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	isTruncated := false
	var nextToken string

	if len(objects) > maxKeys {
		isTruncated = true
		objects = objects[:maxKeys]                   // Trim to requested limit
		nextToken = objects[len(objects)-1].ObjectKey // Use last object id as continuation token
	}

	return &ListObjectsV2Result{
		Objects:               objects,
		CommonPrefixes:        nil, // Not implemented yet
		NextContinuationToken: nextToken,
		IsTruncated:           isTruncated,
	}, nil
}

func (s *service) DeleteObject(ctx context.Context, bucketId int, objectKey string) ([]string, error) {
	rows, err := s.db.Query(ctx, `
		WITH files AS (
			SELECT s.file_id
			FROM objects o JOIN object_segments s ON s.object_id = o.object_id
			WHERE o.bucket_id = $1 AND o.object_key = $2
		), del AS (
			DELETE FROM objects WHERE bucket_id = $1 AND object_key = $2
		)
		SELECT file_id FROM files`, bucketId, objectKey)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
