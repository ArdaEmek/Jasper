package bucket

import (
	"context"
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/server/utils"
)

type Handler struct {
	db database.Service
}

func New() *Handler {
	db := database.New()
	return &Handler{db: db}
}

func (h *Handler) RegisterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("GET /{bucket}/{key...}", h.middleware(h.getObjectHandler))
	mux.HandleFunc("PUT /{bucket}/{key...}", h.putRouter)
	mux.HandleFunc("POST /{bucket}/{key...}", h.multipartRouter)

	mux.HandleFunc("PUT /{bucket}", h.putBucketHandler)
}

func (h *Handler) multipartRouter(w http.ResponseWriter, r *http.Request) {
	// /bucket/key?uploads
	if r.URL.Query().Has("uploads") {
		h.middleware(h.createMultipartUploadHandler)(w, r)
		return
	}

	utils.S3ErrorResponse(w, utils.S3Error{
		Code:     "NotImplemented",
		Message:  "The functionality you requested is not implemented",
		Resource: r.URL.Path,
	})
}

// Helper function for route to call putBucketHandler when key is empty.
func (h *Handler) putRouter(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" || key == "/" {
		h.putBucketHandler(w, r)
		return
	}

	if r.URL.Query().Has("partNumber") && r.URL.Query().Has("uploadId") {
		h.middleware(h.uploadPartHandler)(w, r)
		return
	}

	h.middleware(h.putObjectHandler)(w, r)
}

func (h *Handler) middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		var apiKey *database.ApiKey
		var authErr *database.AuthError

		if len(authHeader) > 0 {
			user, apiKey, authErr = h.db.ValidateHeaderAuth(r.Context(), r)
		} else {
			user, apiKey, authErr = h.db.ValidatePresignedUrl(r.Context(), r)
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

		if user == nil || apiKey == nil {
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "AccessDenied",
				Message:   "Access Denied",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}

		// Get Bucket
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

		bucket, err := h.db.GetBucketByName(r.Context(), bucketName)
		if err != nil {
			log.Printf("Database error: %v", err)
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "InternalError",
				Message:   "An internal error occurred. Try again.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}

		if bucket == nil {
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "NoSuchBucket",
				Message:   "The specified bucket does not exist",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}

		if bucket.OwnerId != user.Id {
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "AccessDenied",
				Message:   "Access Denied",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}

		ctx := context.WithValue(r.Context(), "bucket", bucket)
		ctx = context.WithValue(ctx, "user", user)
		ctx = context.WithValue(ctx, "apiKey", apiKey)
		r = r.WithContext(ctx)

		next(w, r)
	}
}
