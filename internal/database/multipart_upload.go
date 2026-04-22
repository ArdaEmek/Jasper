package database

import (
	"context"
	"encoding/json"
	"time"
)

type MultipartUpload struct {
	UploadID           string
	BucketName         string
	ObjectKey          string
	UserID             int
	ContentType        *string
	ContentDisposition *string
	ContentLanguage    *string
	CustomMetadata     map[string]string
	CreatedAt          time.Time
}

type MultipartUploadPart struct {
	PartNumber int
	ETag       string
	Size       int64
	CreatedAt  time.Time
}

func (s *service) CreateMultipartUpload(ctx context.Context, uploadID, bucketName, objectKey string, userID int, contentType, contentDisp, contentLang string, customMeta map[string]string) error {
	inputMetaJSON, err := json.Marshal(customMeta)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO multipart_uploads (
			upload_id, bucket_name, object_key, user_id, 
			content_type, content_disposition, content_language, custom_metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = s.db.Exec(ctx, query, uploadID, bucketName, objectKey, userID, contentType, contentDisp, contentLang, inputMetaJSON)
	return err
}

func (s *service) GetMultipartUpload(ctx context.Context, uploadID string) (*MultipartUpload, error) {
	query := `
		SELECT upload_id, bucket_name, object_key, user_id, 
		content_type, content_disposition, content_language, custom_metadata, created_at
		FROM multipart_uploads 
		WHERE upload_id = $1
	`
	row := s.db.QueryRow(ctx, query, uploadID)

	var mu MultipartUpload
	var customMetaJSON []byte

	err := row.Scan(
		&mu.UploadID, &mu.BucketName, &mu.ObjectKey, &mu.UserID,
		&mu.ContentType, &mu.ContentDisposition, &mu.ContentLanguage,
		&customMetaJSON, &mu.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	mu.CustomMetadata = make(map[string]string)
	if len(customMetaJSON) > 0 {
		if err := json.Unmarshal(customMetaJSON, &mu.CustomMetadata); err != nil {
			return nil, err
		}
	}

	return &mu, nil
}

func (s *service) SaveMultipartUploadPart(ctx context.Context, uploadID string, partNumber int, etag string, size int64) error {
	query := `
		INSERT INTO multipart_upload_parts (upload_id, part_number, etag, size)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (upload_id, part_number) 
		DO UPDATE SET etag = EXCLUDED.etag, size = EXCLUDED.size
	`
	_, err := s.db.Exec(ctx, query, uploadID, partNumber, etag, size)
	return err
}

func (s *service) ListMultipartUploadParts(ctx context.Context, uploadID string, partNumberMarker int, maxParts int) ([]MultipartUploadPart, error) {
	query := `
		SELECT part_number, etag, size, created_at
		FROM multipart_upload_parts
		WHERE upload_id = $1 AND part_number > $2
		ORDER BY part_number ASC
		LIMIT $3
	`
	rows, err := s.db.Query(ctx, query, uploadID, partNumberMarker, maxParts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []MultipartUploadPart
	for rows.Next() {
		var part MultipartUploadPart
		if err := rows.Scan(&part.PartNumber, &part.ETag, &part.Size, &part.CreatedAt); err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return parts, nil
}
