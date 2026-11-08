package bucket

import (
	"fmt"
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/filesystem"
	"s3/internal/server/utils"
	"strconv"
)

func (h *Handler) getObjectHandler(w http.ResponseWriter, r *http.Request) {
	reqID, _ := r.Context().Value("requestID").(string)
	bucket, _ := r.Context().Value("bucket").(*database.Bucket)
	apiKey, _ := r.Context().Value("apiKey").(*database.ApiKey)

	if apiKey.PermissionLevel == "write_only" {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "AccessDenied",
			Message:   "The provided API key has write-only permissions and cannot get resources.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	key := r.PathValue("key")
	if key == "" {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "AccessDenied",
			Message:   "Access Denied",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	objectData, err := h.db.GetObjectByKey(r.Context(), bucket.Id, key)
	if err != nil {
		log.Printf("Database error retrieving object: %v", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if objectData == nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "NoSuchKey",
			Message:   "The specified key does not exist.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if objectData.ContentType != "" {
		w.Header().Set("Content-Type", objectData.ContentType)
	}
	if objectData.ContentDisposition != "" {
		w.Header().Set("Content-Disposition", objectData.ContentDisposition)
	}
	if objectData.ContentLanguage != "" {
		w.Header().Set("Content-Language", objectData.ContentLanguage)
	}
	if objectData.ETag != "" {
		w.Header().Set("ETag", fmt.Sprintf("\"%s\"", objectData.ETag))
	}
	if !objectData.CreatedAt.IsZero() {
		w.Header().Set("Last-Modified", objectData.CreatedAt.UTC().Format(http.TimeFormat))
	}

	w.Header().Set("x-amz-checksum-algorithm", "MD5")
	w.Header().Set("Accept-Ranges", "bytes")

	// Set custom metadata headers
	for metaKey, metaValue := range objectData.CustomMetadata {
		w.Header().Set("x-amz-meta-"+metaKey, metaValue)
	}

	// Return headers if method is HEAD
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", strconv.FormatInt(objectData.SizeBytes, 10))
		w.WriteHeader(http.StatusOK)
		return
	}

	fs := filesystem.GetInstance()
	if fs == nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	dbSegs, err := h.db.GetObjectSegments(r.Context(), objectData.ObjectId)
	if err != nil {
		log.Printf("Database error retrieving segments for %s: %v", objectData.ObjectId, err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	segs := make([]filesystem.Segment, len(dbSegs))
	for i, s := range dbSegs {
		segs[i] = filesystem.Segment{FileID: s.FileID, Size: s.Size, Start: s.Start}
	}
	sr := fs.NewSegmentedReader(segs)

	size := objectData.SizeBytes
	if sr.Size() != size {
		log.Printf("segment size mismatch for %s: segments=%d object=%d", objectData.ObjectId, sr.Size(), size)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	start, length := int64(0), size
	status := http.StatusOK

	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		if s, e, ok := fs.ParseRange(rangeHeader, size); ok {
			if s >= size {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
				utils.S3ErrorResponse(w, utils.S3Error{
					Code:      "InvalidRange",
					Message:   "The requested range is not satisfiable",
					RequestId: reqID,
					Resource:  r.URL.Path,
				})
				return
			}
			start, length = s, e-s+1
			status = http.StatusPartialContent
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", s, e, size))
		}
	}

	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.WriteHeader(status)

	if err := sr.WriteRange(w, start, length); err != nil {
		log.Printf("Error streaming %s to client: %v", objectData.ObjectId, err)
	}
}
