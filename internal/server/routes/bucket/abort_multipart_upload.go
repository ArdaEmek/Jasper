package bucket

import (
	"fmt"
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/filesystem"
	"s3/internal/server/utils"
)

func (h *Handler) abortMultipartUploadHandler(w http.ResponseWriter, r *http.Request) {
	reqID, _ := r.Context().Value("requestID").(string)
	bucket, _ := r.Context().Value("bucket").(*database.Bucket)
	user, _ := r.Context().Value("user").(*database.User)

	key := r.PathValue("key")
	query := r.URL.Query()
	uploadID := query.Get("uploadId")

	if uploadID == "" {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InvalidArgument",
			Message:   "The upload ID query parameter must be specified.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	// 1. Verify the upload exists and belongs to this bucket/key/user
	mu, err := h.db.GetMultipartUpload(r.Context(), uploadID)
	if err != nil || mu == nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "NoSuchUpload",
			Message:   "The specified multipart upload does not exist. The upload ID might not be valid, or the multipart upload might have been aborted or completed.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if mu.BucketName != bucket.Name || mu.ObjectKey != key || mu.UserID != user.Id {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "AccessDenied",
			Message:   "Access Denied",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	// Fetch all parts
	dbParts, err := h.db.ListMultipartUploadParts(r.Context(), uploadID, 0, 10000)
	if err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	err = h.db.DeleteMultipartUpload(r.Context(), uploadID)
	if err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	// Deleting parts
	fs := filesystem.GetInstance()
	for _, part := range dbParts {
		partFileName := fmt.Sprintf("%s_%d", uploadID, part.PartNumber)
		if err := fs.DeleteFile(partFileName); err != nil {
			log.Printf("Warning: failed to delete part file %s on abort: %v", partFileName, err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
