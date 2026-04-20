package bucket

import (
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/filesystem"
	"s3/internal/server/utils"
)

func (h *Handler) delObjectHandler(w http.ResponseWriter, r *http.Request) {
	reqID, _ := r.Context().Value("requestID").(string)
	bucket, _ := r.Context().Value("bucket").(*database.Bucket)
	apiKey, _ := r.Context().Value("apiKey").(*database.ApiKey)

	if apiKey.PermissionLevel == "read_only" {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "AccessDenied",
			Message:   "The provided API key has read-only permissions and cannot modify resources.",
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

	existingObj, err := h.db.GetObjectByKey(r.Context(), bucket.Id, key)
	if err != nil {
		log.Printf("Database error getting object: %v\n", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if existingObj == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err = h.db.DeleteObject(r.Context(), bucket.Id, key)
	if err != nil {
		log.Printf("Database error deleting object metadata: %v\n", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "Failed to delete object",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	fs := filesystem.GetInstance()
	if fs != nil {
		err := fs.DeleteFile(existingObj.ObjectId)
		if err != nil {
			log.Printf("Failed to physically delete file %s: %v\n", existingObj.ObjectId, err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
