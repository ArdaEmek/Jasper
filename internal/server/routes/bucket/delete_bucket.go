package bucket

import (
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/server/utils"
)

func (h *Handler) deleteBucketHandler(w http.ResponseWriter, r *http.Request) {
	reqID, _ := r.Context().Value("requestID").(string)

	// Check auth header
	authHeader := r.Header.Get("Authorization")
	querySignature := r.URL.Query().Get("X-Amz-Signature")
	if authHeader == "" && querySignature == "" {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "AccessDenied",
			Message:   "Access Denied",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	// Validate token
	var user *database.User
	var authErr *database.AuthError

	if len(authHeader) > 0 {
		user, _, authErr = h.db.ValidateHeaderAuth(r.Context(), r)
	} else {
		user, _, authErr = h.db.ValidatePresignedUrl(r.Context(), r)
	}

	if authErr != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      authErr.Code,
			Message:   authErr.Message,
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if user == nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "AccessDenied",
			Message:   "Access Denied",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	bucketName := r.PathValue("bucket")
	if len(bucketName) < 3 || len(bucketName) > 63 {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InvalidBucketName",
			Message:   "The specified bucket is not valid.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	existingBucket, err := h.db.GetBucketByName(r.Context(), bucketName)
	if err != nil {
		log.Printf("Database error checking bucket existence: %v", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if existingBucket == nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "NoSuchBucket",
			Message:   "The specified bucket does not exist.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if existingBucket.OwnerId != user.Id {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "AccessDenied",
			Message:   "Access Denied.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	isEmpty, err := h.db.IsBucketEmpty(r.Context(), existingBucket.Id, bucketName)
	if err != nil {
		log.Printf("Database error deleting bucket: %v", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if !isEmpty {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "BucketNotEmpty",
			Message:   "The bucket that you tried to delete is not empty.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	err = h.db.DeleteBucket(r.Context(), bucketName, user.Id, "us-east-1")
	if err != nil {
		log.Printf("Database error deleting bucket: %v", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	w.Header().Set("Location", "/"+bucketName)
	w.WriteHeader(http.StatusOK)
}
