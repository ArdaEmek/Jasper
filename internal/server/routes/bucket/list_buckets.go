package bucket

import (
	"net/http"
	"s3/internal/database"
	"s3/internal/server/utils"
	"strconv"
)

func (h *Handler) listBucketsHandler(w http.ResponseWriter, r *http.Request) {
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

	buckets, err := h.db.ListBucketsByOwnerId(r.Context(), user.Id)
	if err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	var bucketInfos []utils.BucketInfo
	for _, bucket := range buckets {
		bucketInfos = append(bucketInfos, utils.BucketInfo{
			Name:         bucket.Name,
			CreationDate: bucket.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
			BucketRegion: bucket.Region,
		})
	}

	utils.S3Response(w, utils.ListBucketsResponse{
		Owner: utils.BucketOwner{
			ID:          strconv.Itoa(user.Id),
			DisplayName: user.Name,
		},
		Buckets: utils.BucketList{
			Buckets: bucketInfos,
		},
	})
}
