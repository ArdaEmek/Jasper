package bucket

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/filesystem"
	"s3/internal/server/utils"
	"strings"

	"github.com/klauspost/crc32"

	"github.com/segmentio/ksuid"
)

func (h *Handler) putObjectHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	reqID, _ := r.Context().Value("requestID").(string)
	bucket, _ := r.Context().Value("bucket").(*database.Bucket)

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

	contentLanguage := r.Header.Get("Content-Language")
	contentDisposition := r.Header.Get("Content-Disposition")
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	metadataSize := len(contentType) + len(contentLanguage) + len(contentDisposition)
	if metadataSize > 2048 {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "MetadataTooLarge",
			Message:   "Your system metadata exceeds the maximum allowed size (2 KB)",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	customMetadata := make(map[string]string)
	customMetadataSize := 0
	for key, val := range r.Header {
		if strings.HasPrefix(key, "X-Amz-Meta-") && len(val) > 0 {
			customMetadata[key] = val[0]
			customMetadataSize += len(key) + len(val[0])
		}
	}

	if customMetadataSize > 2048 {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "MetadataTooLarge",
			Message:   "Your user metadata exceeds the maximum allowed size (2 KB)",
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

	// Check if this is an overwrite so we can clean up the old file later
	existingObj, _ := h.db.GetObjectByKey(r.Context(), bucket.Id, key)

	objectID := ksuid.New().String()
	fw, err := fs.WriteFile(objectID)
	if err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "Failed to create object",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}
	defer fw.Close()

	hash := md5.New()
	CRC32Hash := crc32.NewIEEE()
	multiWriter := io.MultiWriter(fw.Writer, hash, CRC32Hash)

	sizeBytes, err := io.Copy(multiWriter, r.Body)
	if err != nil {
		fw.Close()
		_ = fs.DeleteFile(objectID)
		log.Printf("Error writing file: %v\n", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "Failed to write object",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	fw.Close()
	etag := hex.EncodeToString(hash.Sum(nil))

	serverCRC32 := base64.StdEncoding.EncodeToString(CRC32Hash.Sum(nil))
	clientCRC32 := query.Get("x-amz-checksum-crc32")
	if clientCRC32 != "" && clientCRC32 != serverCRC32 {
		_ = fs.DeleteFile(objectID)
		log.Printf("CRC32 mismatch: expected %s, got %s", clientCRC32, serverCRC32)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InvalidDigest",
			Message:   "The CRC32 checksum of the object does not match",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	_, err = h.db.CreateObject(r.Context(), bucket.Id, objectID, key, sizeBytes, contentType, etag, contentDisposition, contentLanguage, customMetadata)
	if err != nil {
		_ = fs.DeleteFile(objectID)
		log.Printf("Database error creating object: %v\n", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "Failed to save object metadata",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	// Delete the old object with the same key
	if existingObj != nil {
		err := fs.DeleteFile(existingObj.ObjectId)
		if err != nil {
			log.Printf("Failed to delete overwritten file %s: %v\n", existingObj.ObjectId, err)
		}
	}

	w.Header().Set("ETag", fmt.Sprintf("\"%s\"", etag))
	w.Header().Set("X-Amz-Checksum-Crc32", serverCRC32)
	w.WriteHeader(http.StatusOK)
}
