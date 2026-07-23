package bucket

import (
	"net/http"
	"s3/internal/database"
	"s3/internal/server/utils"
	"strconv"
)

func (h *Handler) listObjectsHandler(w http.ResponseWriter, r *http.Request) {
	reqID, _ := r.Context().Value("requestID").(string)
	bucket, _ := r.Context().Value("bucket").(*database.Bucket)
	user, _ := r.Context().Value("user").(*database.User)

	query := r.URL.Query()

	maxKeys := 1000
	if mkStr := query.Get("max-keys"); mkStr != "" {
		if mk, err := strconv.Atoi(mkStr); err == nil && mk >= 0 && mk <= 1000 {
			maxKeys = mk
		}
	}

	prefix := query.Get("prefix")
	continuationToken := query.Get("continuation-token")
	delimiter := query.Get("delimiter")
	startAfter := query.Get("start-after")
	fetchOwner := query.Get("fetch-owner") == "true"

	res, err := h.db.ListObjectsV2(
		r.Context(),
		bucket.Id,
		prefix,
		continuationToken,
		startAfter,
		maxKeys,
	)
	if err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	// Creating the XML response
	var contents []utils.ObjectItemResponse
	for _, obj := range res.Objects {
		item := utils.ObjectItemResponse{
			Key:          obj.ObjectKey,
			LastModified: obj.CreatedAt.UTC(),
			ETag:         `"` + obj.ETag + `"`,
			Size:         obj.SizeBytes,
		}

		if fetchOwner && user != nil {
			item.Owner = user.Id
		}

		contents = append(contents, item)
	}

	// Map common prefixes (folders)
	var commonPrefixes []utils.CommonPrefixResponse
	for _, p := range res.CommonPrefixes {
		commonPrefixes = append(commonPrefixes, utils.CommonPrefixResponse{
			Prefix: p,
		})
	}

	utils.S3Response(w, utils.ListBucketResultV2Response{
		Name:                  bucket.Name,
		Prefix:                prefix,
		MaxKeys:               maxKeys,
		KeyCount:              len(contents) + len(commonPrefixes),
		Delimiter:             delimiter,
		IsTruncated:           res.IsTruncated,
		ContinuationToken:     continuationToken,
		NextContinuationToken: res.NextContinuationToken,
		StartAfter:            startAfter,
		Contents:              contents,
		CommonPrefixes:        commonPrefixes,
	})
}
