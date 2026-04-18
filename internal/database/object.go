package database

import (
	"context"
	"encoding/json"
	"time"
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
