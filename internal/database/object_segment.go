package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type ObjectSegment struct {
	FileID string
	Size   int64
	Start  int64
}

func (s *service) GetObjectSegments(ctx context.Context, objectID string) ([]ObjectSegment, error) {
	rows, err := s.db.Query(ctx,
		`SELECT file_id, size, start_offset FROM object_segments WHERE object_id = $1 ORDER BY idx`, objectID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (ObjectSegment, error) {
		var sg ObjectSegment
		err := r.Scan(&sg.FileID, &sg.Size, &sg.Start)
		return sg, err
	})
}

func (s *service) ReplaceObject(ctx context.Context, o Object, segs []ObjectSegment) ([]string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	old, err := replaceObjectTx(ctx, tx, o, segs)
	if err != nil {
		return nil, err
	}
	return old, tx.Commit(ctx)
}

// replaceObjectTx deletes any object at the key and inserts the new one with
// its segments. It returns the replaced object's file IDs.
func replaceObjectTx(ctx context.Context, tx pgx.Tx, o Object, segs []ObjectSegment) ([]string, error) {
	var sum int64
	for _, sg := range segs {
		sum += sg.Size
	}
	if sum != o.SizeBytes {
		return nil, fmt.Errorf("segment sizes (%d) do not match object size (%d)", sum, o.SizeBytes)
	}

	// serialize writers of the same key, even when there is no old row to lock
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		fmt.Sprintf("%d/%s", o.BucketId, o.ObjectKey)); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT s.file_id
		FROM objects o JOIN object_segments s ON s.object_id = o.object_id
		WHERE o.bucket_id = $1 AND o.object_key = $2`, o.BucketId, o.ObjectKey)
	if err != nil {
		return nil, err
	}
	oldFiles, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	// cascades to the old object's segment rows
	if _, err := tx.Exec(ctx, `DELETE FROM objects WHERE bucket_id = $1 AND object_key = $2`,
		o.BucketId, o.ObjectKey); err != nil {
		return nil, err
	}

	metaJSON, err := json.Marshal(o.CustomMetadata)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO objects (object_id, bucket_id, object_key, size_bytes, etag,
                     content_type, content_disposition, content_language, custom_metadata, is_public)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		o.ObjectId, o.BucketId, o.ObjectKey, o.SizeBytes, o.ETag,
		o.ContentType, o.ContentDisposition, o.ContentLanguage, metaJSON, o.IsPublic); err != nil {
		return nil, err
	}

	idx := make([]int32, len(segs))
	fileIDs := make([]string, len(segs))
	sizes := make([]int64, len(segs))
	starts := make([]int64, len(segs))
	for i, sg := range segs {
		idx[i], fileIDs[i], sizes[i], starts[i] = int32(i), sg.FileID, sg.Size, sg.Start
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO object_segments (object_id, idx, file_id, size, start_offset)
		SELECT $1, t.idx, t.file_id, t.size, t.start
		FROM unnest($2::int[], $3::text[], $4::bigint[], $5::bigint[]) AS t(idx, file_id, size, start)`,
		o.ObjectId, idx, fileIDs, sizes, starts); err != nil {
		return nil, err
	}
	return oldFiles, nil
}
