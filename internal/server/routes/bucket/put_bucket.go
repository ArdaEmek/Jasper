package bucket

import (
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/server/utils"
)

func (h *Handler) putBucketHandler(w http.ResponseWriter, r *http.Request) {
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
		user, _, authErr = h.db.ValidatePresignedUrl(r.Context(), r.URL.Query())
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

	if existingBucket != nil {
		if existingBucket.OwnerId == user.Id {
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "BucketAlreadyOwnedByYou",
				Message:   "Your previous request to create the named bucket succeeded and you already own it.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
		} else {
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "BucketAlreadyExists",
				Message:   "The requested bucket name is not available. The bucket namespace is shared by all users of the system.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
		}
		return
	}

	_, err = h.db.CreateBucket(r.Context(), bucketName, user.Id, "us-east-1")
	if err != nil {
		log.Printf("Database error creating bucket: %v", err)
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
