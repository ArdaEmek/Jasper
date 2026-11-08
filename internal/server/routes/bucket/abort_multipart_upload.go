package bucket

import (
	"errors"
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/filesystem"
	"s3/internal/server/utils"

	"github.com/jackc/pgx/v5"
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
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "NoSuchUpload",
				Message:   "The specified multipart upload does not exist. The upload ID might not be valid, or the multipart upload might have been aborted or completed.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}
		log.Printf("get multipart upload %s: %v", uploadID, err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
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

	files, err := h.db.DeleteMultipartUpload(r.Context(), uploadID)
	if err != nil {
		log.Printf("delete multipart upload %s: %v", uploadID, err)
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
	if err := fs.DeleteFiles(files); err != nil {
		log.Printf("cleanup of aborted upload %s: %v", uploadID, err)
	}

	w.WriteHeader(http.StatusNoContent)
}
