package bucket

import (
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/server/utils"
	"strings"

	"github.com/segmentio/ksuid"
)

func (h *Handler) createMultipartUploadHandler(w http.ResponseWriter, r *http.Request) {
	reqID, _ := r.Context().Value("requestID").(string)
	bucket, _ := r.Context().Value("bucket").(*database.Bucket)
	apiKey, _ := r.Context().Value("apiKey").(*database.ApiKey)

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

	contentLanguage := r.Header.Get("Content-Language")
	contentDisposition := r.Header.Get("Content-Disposition")
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	metadataSize := len(contentType) + len(contentLanguage) + len(contentDisposition)
	if metadataSize > 2048 {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "MetadataTooLarge",
			Message:   "Your system metadata exceeds the maximum allowed size (2 KB)",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	isPublic := false
	acl := r.Header.Get("X-Amz-Acl")
	if acl == "" {
		acl = r.URL.Query().Get("x-amz-acl")
	}

	switch acl {
	case "", "private":
	case "public-read":
		isPublic = true
	default:
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "NotImplemented",
			Message:   "Only the private and public-read ACLs are supported",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	customMetadata := make(map[string]string)
	customMetadataSize := 0
	const metaPrefix = "X-Amz-Meta-"
	for key, val := range r.Header {
		if strings.HasPrefix(key, metaPrefix) && len(val) > 0 {
			k := strings.ToLower(key[len(metaPrefix):])
			customMetadata[k] = val[0]
			customMetadataSize += len(k) + len(val[0])
		}
	}

	if customMetadataSize > 2048 {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "MetadataTooLarge",
			Message:   "Your user metadata exceeds the maximum allowed size (2 KB)",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if len(key) > 1024 {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "KeyTooLongError",
			Message:   "Your key is too long.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	uploadID := ksuid.New().String()
	err := h.db.CreateMultipartUpload(
		r.Context(),
		uploadID,
		bucket.Name,
		key,
		apiKey.UserId,
		contentType,
		contentDisposition,
		contentLanguage,
		customMetadata,
		isPublic,
	)
	if err != nil {
		log.Printf("Failed to create multipart upload DB entry: %v", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	utils.S3Response(w, utils.CreateMultipartUploadResponse{
		Bucket:   bucket.Name,
		Key:      key,
		UploadId: uploadID,
	})
}
