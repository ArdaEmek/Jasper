package database

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Object struct {
	ObjectId           string            `json:"object_id"`
	BucketId           int               `json:"bucket_id"`
	ObjectKey          string            `json:"object_key"`
	SizeBytes          int64             `json:"size_bytes"`
	ContentType        string            `json:"content_type"`
	ContentDisposition string            `json:"content_disposition"`
	ContentLanguage    string            `json:"content_language"`
	CustomMetadata     map[string]string `json:"custom_metadata"`
	ETag               string            `json:"etag"`
	CreatedAt          time.Time         `json:"created_at"`
}

func (s *service) CreateObject(
	ctx context.Context,
	bucketId int,
	objectId, objectKey string,
	sizeBytes int64,
	contentType, etag, contentDisp, contentLang string,
	customMeta map[string]string,
) (*Object, error) {
	inputMetaJSON, err := json.Marshal(customMeta)
	if err != nil {
		return nil, err
	}

	query := `
       INSERT INTO objects (
           object_id, bucket_id, object_key, size_bytes, content_type, etag,
           content_disposition, content_language, custom_metadata
       )
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
       ON CONFLICT (bucket_id, object_key) DO UPDATE
       SET size_bytes = EXCLUDED.size_bytes,
           content_type = EXCLUDED.content_type,
           etag = EXCLUDED.etag,
           content_disposition = EXCLUDED.content_disposition,
           content_language = EXCLUDED.content_language,
           custom_metadata = EXCLUDED.custom_metadata,
           created_at = CURRENT_TIMESTAMP,
           object_id = EXCLUDED.object_id
       RETURNING 
           object_id, bucket_id, object_key, size_bytes, content_type, etag,
           content_disposition, content_language, created_at
    `
	var obj Object
	obj.CustomMetadata = customMeta

	err = s.db.QueryRow(ctx, query,
		objectId, bucketId, objectKey, sizeBytes, contentType, etag,
		contentDisp, contentLang, inputMetaJSON,
	).Scan(
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

	return &obj, nil
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

func (s *service) DeleteObject(ctx context.Context, bucketId int, objectKey string) error {
	query := `DELETE FROM objects WHERE bucket_id = $1 AND object_key = $2`
	_, err := s.db.Exec(ctx, query, bucketId, objectKey)
	return err
}
