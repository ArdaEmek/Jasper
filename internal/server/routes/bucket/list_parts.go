package bucket

import (
	"net/http"
	"s3/internal/database"
	"s3/internal/server/utils"
	"strconv"
)

func (h *Handler) listPartsHandler(w http.ResponseWriter, r *http.Request) {
	reqID, _ := r.Context().Value("requestID").(string)
	bucket, _ := r.Context().Value("bucket").(*database.Bucket)
	user, _ := r.Context().Value("user").(*database.User)

	key := r.PathValue("key")
	query := r.URL.Query()
	uploadID := query.Get("uploadId")

	maxParts := 1000
	if mpStr := query.Get("max-parts"); mpStr != "" {
		if mp, err := strconv.Atoi(mpStr); err == nil && mp > 0 && mp <= 1000 {
			maxParts = mp
		}
	}

	partNumberMarker := 0
	if pnmStr := query.Get("part-number-marker"); pnmStr != "" {
		if pnm, err := strconv.Atoi(pnmStr); err == nil && pnm >= 0 {
			partNumberMarker = pnm
		}
	}

	mu, err := h.db.GetMultipartUpload(r.Context(), uploadID)
	if err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "NoSuchUpload",
			Message:   "The specified upload does not exist. The upload ID may be invalid, or the upload may have been aborted or completed.",
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

	// Fetch maxParts + 1 to determine if truncated
	dbParts, err := h.db.ListMultipartUploadParts(r.Context(), uploadID, partNumberMarker, maxParts+1)
	if err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "Failed to retrieve parts",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	isTruncated := false
	if len(dbParts) > maxParts {
		isTruncated = true
		dbParts = dbParts[:maxParts]
	}

	var parts []database.MultipartUploadPart
	for _, p := range dbParts {
		parts = append(parts, database.MultipartUploadPart{
			PartNumber: p.PartNumber,
			ETag:       `"` + p.ETag + `"`,
			Size:       p.Size,
			CreatedAt:  p.CreatedAt.UTC(),
		})
	}

	nextPartNumberMarker := 0
	if len(parts) > 0 && isTruncated {
		nextPartNumberMarker = parts[len(parts)-1].PartNumber
	}

	utils.S3Response(w, utils.ListPartsResponse{
		Bucket:               bucket.Name,
		Key:                  key,
		UploadId:             uploadID,
		PartNumberMarker:     partNumberMarker,
		NextPartNumberMarker: nextPartNumberMarker,
		MaxParts:             maxParts,
		IsTruncated:          isTruncated,
		Parts:                parts,
	})
}
