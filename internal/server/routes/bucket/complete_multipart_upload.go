package bucket

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
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
	body := http.MaxBytesReader(w, r.Body, 2<<20) // 2MB
	decoder := xml.NewDecoder(body)
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

	dbPartsMap := make(map[int]database.MultipartUploadPart, len(dbParts))
	for _, p := range dbParts {
		dbPartsMap[p.PartNumber] = p
	}

	// Validate parts
	var totalSize int64
	for i, reqPart := range completeReq.Parts {
		if i > 0 && reqPart.PartNumber <= completeReq.Parts[i-1].PartNumber {
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "InvalidPartOrder",
				Message:   "The list of parts was not in ascending order. The parts list must be specified in order by part number.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}

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
		isLastPart := i == len(completeReq.Parts)-1
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

	segs := make([]database.ObjectSegment, 0, len(completeReq.Parts))
	used := make(map[int]string, len(completeReq.Parts))
	hash := md5.New()
	var start int64
	for _, rp := range completeReq.Parts {
		p := dbPartsMap[rp.PartNumber]
		segs = append(segs, database.ObjectSegment{FileID: p.FileID, Size: p.Size, Start: start})
		used[p.PartNumber] = p.FileID
		start += p.Size

		raw, err := hex.DecodeString(p.ETag)
		if err != nil {
			log.Printf("bad stored etag for part %d: %v", p.PartNumber, err)
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "InternalError",
				Message:   "An internal error occurred. Try again.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}
		hash.Write(raw)
	}

	finalEtag := fmt.Sprintf("%s-%d", hex.EncodeToString(hash.Sum(nil)), len(completeReq.Parts))
	contentType := "application/octet-stream"
	if mu.ContentType != nil && *mu.ContentType != "" {
		contentType = *mu.ContentType
	}
	contentDisp, contentLang := "", ""
	if mu.ContentDisposition != nil {
		contentDisp = *mu.ContentDisposition
	}
	if mu.ContentLanguage != nil {
		contentLang = *mu.ContentLanguage
	}

	replacedFiles, unused, err := h.db.CompleteMultipartUpload(r.Context(), uploadID, used, database.Object{
		ObjectId:           ksuid.New().String(),
		BucketId:           bucket.Id,
		ObjectKey:          key,
		SizeBytes:          totalSize,
		ContentType:        contentType,
		ContentDisposition: contentDisp,
		ContentLanguage:    contentLang,
		CustomMetadata:     mu.CustomMetadata,
		ETag:               finalEtag,
	}, segs)

	switch {
	case errors.Is(err, database.ErrNoSuchUpload):
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "NoSuchUpload",
			Message:   "The specified multipart upload does not exist. The upload ID might not be valid, or the multipart upload might have been aborted or completed.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	case errors.Is(err, database.ErrPartsChanged):
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InvalidPart",
			Message:   "One or more of the specified parts could not be found.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	case err != nil:
		log.Printf("complete multipart upload: %v", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if err := fs.DeleteFiles(replacedFiles); err != nil {
		log.Printf("cleanup of replaced files: %v", err)
	}
	if err := fs.DeleteFiles(unused); err != nil {
		log.Printf("cleanup of unused files: %v", err)
	}

	locationURL := fmt.Sprintf("http://%s/%s/%s", r.Host, bucket.Name, key)
	utils.S3Response(w, utils.CompleteMultipartUploadResponse{
		Location: locationURL,
		Bucket:   bucket.Name,
		Key:      key,
		ETag:     `"` + finalEtag + `"`,
	})
}
