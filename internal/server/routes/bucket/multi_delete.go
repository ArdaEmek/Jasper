package bucket

import (
	"encoding/xml"
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/filesystem"
	"s3/internal/server/utils"
	"strconv"
	"strings"
)

func (h *Handler) multiDeleteHandler(w http.ResponseWriter, r *http.Request) {
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

	expectedOwner := r.Header.Get("X-Amz-Expected-Bucket-Owner")
	if expectedOwner == "" {
		expectedOwner = r.URL.Query().Get("x-amz-expected-bucket-owner")
	}

	if expectedOwner != "" && expectedOwner != strconv.Itoa(bucket.OwnerId) {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "AccessDenied",
			Message:   "Access Denied",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if !r.URL.Query().Has("delete") {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InvalidArgument",
			Message:   "Missing required query parameter: delete",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	var deleteReq utils.DeleteObjectsRequest
	decoder := xml.NewDecoder(r.Body)
	if err := decoder.Decode(&deleteReq); err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "MalformedXML",
			Message:   "The XML that you provided was not well formed or did not validate against our published schema.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if len(deleteReq.Objects) == 0 {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InvalidArgument",
			Message:   "No objects specified for deletion.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	fs := filesystem.GetInstance()
	if fs == nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	var deleted []utils.DeletedObject
	var errors []utils.DeleteError

	for _, obj := range deleteReq.Objects {
		if obj.Key == "" {
			errors = append(errors, utils.DeleteError{
				Key:     obj.Key,
				Code:    "InvalidArgument",
				Message: "Object key is required.",
			})
			continue
		}

		existingObj, err := h.db.GetObjectByKey(r.Context(), bucket.Id, obj.Key)
		if err != nil {
			log.Printf("Database error getting object: %v\n", err)
			errors = append(errors, utils.DeleteError{
				Key:     obj.Key,
				Code:    "InternalError",
				Message: "An internal error occurred. Try again.",
			})
			continue
		}

		if existingObj == nil {
			if !deleteReq.Quiet {
				deleted = append(deleted, utils.DeletedObject{Key: obj.Key})
			}
			continue
		}

		if obj.ETag != "" {
			// Trimming quotes from both request and database ETag, it'll be compatible if request doesn't have it.
			reqETag := strings.Trim(obj.ETag, `"`)
			dbETag := strings.Trim(existingObj.ETag, `"`)

			if reqETag != dbETag {
				errors = append(errors, utils.DeleteError{
					Key:     obj.Key,
					Code:    "PreconditionFailed",
					Message: "At least one of the preconditions that you specified did not hold.",
				})
				continue
			}
		}

		if obj.Size != nil {
			if existingObj.SizeBytes != *obj.Size {
				errors = append(errors, utils.DeleteError{
					Key:     obj.Key,
					Code:    "PreconditionFailed",
					Message: "At least one of the preconditions that you specified did not hold.",
				})
				continue
			}
		}

		err = h.db.DeleteObject(r.Context(), bucket.Id, obj.Key)
		if err != nil {
			log.Printf("Database error deleting object metadata: %v\n", err)
			errors = append(errors, utils.DeleteError{
				Key:     obj.Key,
				Code:    "InternalError",
				Message: "Failed to delete object",
			})
			continue
		}

		if err := fs.DeleteFile(existingObj.ObjectId); err != nil {
			log.Printf("Failed to physically delete file %s: %v\n", existingObj.ObjectId, err)
		}

		if !deleteReq.Quiet {
			deleted = append(deleted, utils.DeletedObject{Key: obj.Key})
		}
	}

	utils.S3Response(w, utils.DeleteObjectsResult{
		Deleted: deleted,
		Errors:  errors,
	})
}
