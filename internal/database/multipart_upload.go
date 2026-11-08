package database

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
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
	IsPublic           bool
}

type MultipartUploadPart struct {
	PartNumber int
	ETag       string
	Size       int64
	FileID     string
	CreatedAt  time.Time
}

var (
	ErrNoSuchUpload = errors.New("no such upload")
	ErrPartsChanged = errors.New("parts changed during completion")
)

func (s *service) CreateMultipartUpload(ctx context.Context, uploadID, bucketName, objectKey string, userID int, contentType, contentDisp, contentLang string, customMeta map[string]string, isPublic bool) error {
	inputMetaJSON, err := json.Marshal(customMeta)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO multipart_uploads (
			upload_id, bucket_name, object_key, user_id, 
			content_type, content_disposition, content_language, custom_metadata, is_public
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = s.db.Exec(ctx, query, uploadID, bucketName, objectKey, userID, contentType, contentDisp, contentLang, inputMetaJSON, isPublic)
	return err
}

func (s *service) GetMultipartUpload(ctx context.Context, uploadID string) (*MultipartUpload, error) {
	query := `
		SELECT upload_id, bucket_name, object_key, user_id, 
		content_type, content_disposition, content_language, custom_metadata, created_at, is_public
		FROM multipart_uploads 
		WHERE upload_id = $1
	`
	row := s.db.QueryRow(ctx, query, uploadID)

	var mu MultipartUpload
	var customMetaJSON []byte

	err := row.Scan(
		&mu.UploadID, &mu.BucketName, &mu.ObjectKey, &mu.UserID,
		&mu.ContentType, &mu.ContentDisposition, &mu.ContentLanguage,
		&customMetaJSON, &mu.CreatedAt, &mu.IsPublic,
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

func (s *service) SaveMultipartUploadPart(ctx context.Context, uploadID string, partNumber int, etag string, size int64, fileID string) (*string, error) {
	query := `
		INSERT INTO multipart_upload_parts (upload_id, part_number, etag, size, file_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (upload_id, part_number) 
		DO UPDATE SET etag = EXCLUDED.etag, size = EXCLUDED.size, file_id = EXCLUDED.file_id
		RETURNING (SELECT file_id FROM multipart_upload_parts
                   WHERE upload_id = $1 AND part_number = $2)
	`

	var oldFileID *string
	err := s.db.QueryRow(ctx, query, uploadID, partNumber, etag, size, fileID).Scan(&oldFileID)
	return oldFileID, err
}

func (s *service) CompleteMultipartUpload(ctx context.Context, uploadID string, used map[int]string,
	o Object, segs []ObjectSegment) (oldFiles, unused []string, err error) {

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	var one int
	err = tx.QueryRow(ctx, `SELECT 1 FROM multipart_uploads WHERE upload_id = $1 FOR UPDATE`, uploadID).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNoSuchUpload
	}
	if err != nil {
		return nil, nil, err
	}

	// lock the part rows so a concurrent re-upload can't swap a file under us
	rows, err := tx.Query(ctx,
		`SELECT part_number, file_id FROM multipart_upload_parts WHERE upload_id = $1 FOR UPDATE`, uploadID)
	if err != nil {
		return nil, nil, err
	}
	seen := 0
	for rows.Next() {
		var pn int
		var fid string
		if err := rows.Scan(&pn, &fid); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if want, ok := used[pn]; ok {
			if want != fid {
				rows.Close()
				return nil, nil, ErrPartsChanged
			}
			seen++
		} else {
			unused = append(unused, fid)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if seen != len(used) {
		return nil, nil, ErrPartsChanged
	}

	oldFiles, err = replaceObjectTx(ctx, tx, o, segs)
	if err != nil {
		return nil, nil, err
	}

	// cascades to the part rows; their files now back the object or are in `unused`
	if _, err := tx.Exec(ctx, `DELETE FROM multipart_uploads WHERE upload_id = $1`, uploadID); err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return oldFiles, unused, nil
}

func (s *service) ListMultipartUploadParts(ctx context.Context, uploadID string, partNumberMarker int, maxParts int) ([]MultipartUploadPart, error) {
	query := `
		SELECT part_number, etag, size, file_id, created_at
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
		if err := rows.Scan(&part.PartNumber, &part.ETag, &part.Size, &part.FileID, &part.CreatedAt); err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return parts, nil
}

func (s *service) DeleteMultipartUpload(ctx context.Context, uploadID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `
		WITH files AS (
			SELECT file_id FROM multipart_upload_parts WHERE upload_id = $1
		), del AS (
			DELETE FROM multipart_uploads WHERE upload_id = $1
		)
		SELECT file_id FROM files`, uploadID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
