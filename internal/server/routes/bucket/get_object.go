package bucket

import (
	"fmt"
	"io"
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

	fr, err := fs.ReadFile(objectData.ObjectId)
	if err != nil {
		log.Printf("Filesystem error retrieving object file %s: %v", objectData.ObjectId, err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}
	defer fr.Close()

	start, end := int64(0), objectData.SizeBytes-1
	isRange := false

	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		n, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
		if err == nil || n > 0 {
			if n == 1 {
				end = objectData.SizeBytes - 1
			}

			if start < 0 || start >= objectData.SizeBytes || start > end {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", objectData.SizeBytes))
				utils.S3ErrorResponse(w, utils.S3Error{
					Code:      "InvalidRange",
					Message:   "The requested range cannot be satisfied.",
					RequestId: reqID,
					Resource:  r.URL.Path,
				})
				return
			}

			if end >= objectData.SizeBytes {
				end = objectData.SizeBytes - 1
			}
			isRange = true
		}
	}

	if isRange {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, objectData.SizeBytes))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)

		if _, err := fr.File.Seek(start, io.SeekStart); err != nil {
			log.Printf("Failed to seek file: %v", err)
		}

		limitedReader := io.LimitReader(fr, end-start+1)
		if _, err := io.Copy(w, limitedReader); err != nil {
			log.Printf("Error streaming partial file %s to client: %v", objectData.ObjectId, err)
		}
	} else {
		w.Header().Set("Content-Length", strconv.FormatInt(objectData.SizeBytes, 10))
		w.WriteHeader(http.StatusOK)

		limitedReader := io.LimitReader(fr, objectData.SizeBytes)
		if _, err := io.Copy(w, limitedReader); err != nil {
			log.Printf("Error streaming file %s to client: %v", objectData.ObjectId, err)
		}
	}
}
