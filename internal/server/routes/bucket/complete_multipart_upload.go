package bucket

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/filesystem"
	"s3/internal/server/utils"

	"github.com/segmentio/ksuid"
)

func (h *Handler) completeMultipartUploadHandler(w http.ResponseWriter, r *http.Request) {
	reqID, _ := r.Context().Value("requestID").(string)
	bucket, _ := r.Context().Value("bucket").(*database.Bucket)
	user, _ := r.Context().Value("user").(*database.User)

	key := r.PathValue("key")
	query := r.URL.Query()
	uploadID := query.Get("uploadId")

	// Parse XML Payload
	var completeReq utils.CompleteMultipartUploadRequest
	decoder := xml.NewDecoder(r.Body)
	if err := decoder.Decode(&completeReq); err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "MalformedXML",
			Message:   "The XML that you provided was not well formed or did not validate against our published schema.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	mu, err := h.db.GetMultipartUpload(r.Context(), uploadID)
	if err != nil {
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

	dbPartsMap := make(map[int]database.MultipartUploadPart)
	for _, p := range dbParts {
		dbPartsMap[p.PartNumber] = p
	}

	// Validate parts
	var totalSize int64
	for i, reqPart := range completeReq.Parts {
		dbPart, exists := dbPartsMap[reqPart.PartNumber]
		if !exists {
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "InvalidPart",
				Message:   "One or more of the specified parts could not be found.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}

		// Stripping quotes from ETag
		cleanReqETag := reqPart.ETag
		if len(cleanReqETag) >= 2 && cleanReqETag[0] == '"' && cleanReqETag[len(cleanReqETag)-1] == '"' {
			cleanReqETag = cleanReqETag[1 : len(cleanReqETag)-1]
		}

		if cleanReqETag != dbPart.ETag {
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "InvalidPart",
				Message:   "The part ETag does not match what was uploaded.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}

		// Size check, parts must be > 5MB except for the last part
		isLastPart := (i == len(completeReq.Parts)-1)
		if !isLastPart && dbPart.Size < 5*1024*1024 {
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "EntityTooSmall",
				Message:   "Your proposed upload is smaller than the minimum allowed object size.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}

		totalSize += dbPart.Size
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

	newObjectID := ksuid.New().String()
	finalFile, err := fs.WriteFile(newObjectID)
	if err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	// Putting the parts together
	// ! TEMPORARY, SUBJECT TO CHANGE
	hash := md5.New()
	for _, reqPart := range completeReq.Parts {
		partFileName := fmt.Sprintf("%s_%d", uploadID, reqPart.PartNumber)

		// Decode string ETag
		rawPartHash, _ := hex.DecodeString(dbPartsMap[reqPart.PartNumber].ETag)
		hash.Write(rawPartHash)

		// Read part and write it to object
		partFile, err := fs.ReadFile(partFileName)
		if err != nil {
			finalFile.Close()
			fs.DeleteFile(newObjectID)
			log.Printf("Error reading part file %s: %v", partFileName, err)
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "InternalError",
				Message:   "An internal error occurred. Try again.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}

		_, err = io.Copy(finalFile.File, partFile.File) // Passing *os.File directly to use syscall copy_file_range, bypassing buffering completely
		partFile.Close()
		if err != nil {
			finalFile.Close()
			fs.DeleteFile(newObjectID)
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "InternalError",
				Message:   "An internal error occurred. Try again.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}
	}
	finalFile.Close()

	finalEtag := fmt.Sprintf("%s-%d", hex.EncodeToString(hash.Sum(nil)), len(completeReq.Parts))

	contentDisp := ""
	if mu.ContentDisposition != nil {
		contentDisp = *mu.ContentDisposition
	}
	contentLang := ""
	if mu.ContentLanguage != nil {
		contentLang = *mu.ContentLanguage
	}
	contentType := "application/octet-stream"
	if mu.ContentType != nil {
		contentType = *mu.ContentType
	}

	_, err = h.db.CreateObject(
		r.Context(),
		bucket.Id,
		newObjectID,
		key,
		totalSize,
		contentType,
		finalEtag,
		contentDisp,
		contentLang,
		mu.CustomMetadata,
	)

	if err != nil {
		_ = fs.DeleteFile(newObjectID)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	// Delete old parts
	_ = h.db.DeleteMultipartUpload(r.Context(), uploadID)
	for _, reqPart := range completeReq.Parts {
		_ = fs.DeleteFile(fmt.Sprintf("%s_%d", uploadID, reqPart.PartNumber))
	}

	locationURL := fmt.Sprintf("http://%s/%s/%s", r.Host, bucket.Name, key)
	utils.S3Response(w, utils.CompleteMultipartUploadResponse{
		Location: locationURL,
		Bucket:   bucket.Name,
		Key:      key,
		ETag:     `"` + finalEtag + `"`,
	})
}
