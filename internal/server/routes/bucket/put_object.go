package bucket

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
	const metaPrefix = "X-Amz-Meta-"
	for key, val := range r.Header {
		if strings.HasPrefix(key, metaPrefix) && len(val) > 0 {
			k := strings.ToLower(key[len(metaPrefix):])
			customMetadata[k] = val[0]
			customMetadataSize += len(k) + len(val[0])
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

	objectID := ksuid.New().String()
	fw, err := fs.WriteFile(objectID)
	if err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	defer fw.Abort()

	hash := md5.New()
	CRC32Hash := crc32.NewIEEE()
	multiWriter := io.MultiWriter(fw, hash, CRC32Hash)

	sizeBytes, err := io.Copy(multiWriter, r.Body)
	if err != nil {
		log.Printf("Error writing file: %v\n", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if err := fw.Close(); err != nil {
		log.Printf("Error closing file: %v\n", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	etag := hex.EncodeToString(hash.Sum(nil))
	if md5Header := r.Header.Get("Content-MD5"); md5Header != "" {
		clientMD5, err := base64.StdEncoding.DecodeString(md5Header)
		if err != nil || len(clientMD5) != md5.Size {
			_ = fs.DeleteFile(objectID)
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "InvalidDigest",
				Message:   "The Content-MD5 or checksum value that you specified is not valid.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}
		if !bytes.Equal(clientMD5, hash.Sum(nil)) {
			_ = fs.DeleteFile(objectID)
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "BadDigest",
				Message:   "The Content-MD5 you specified did not match what we received.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}
	}

	serverCRC32 := base64.StdEncoding.EncodeToString(CRC32Hash.Sum(nil))
	clientCRC32 := r.Header.Get("X-Amz-Checksum-Crc32")
	if clientCRC32 == "" {
		clientCRC32 = query.Get("x-amz-checksum-crc32")
	}

	if clientCRC32 != "" && clientCRC32 != serverCRC32 {
		_ = fs.DeleteFile(objectID)
		log.Printf("CRC32 mismatch: expected %s, got %s", clientCRC32, serverCRC32)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "BadDigest",
			Message:   "The CRC32 checksum you specified did not match the calculated checksum.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	// Check if this is an overwrite so we can clean up the old file
	existingObj, err := h.db.GetObjectByKey(r.Context(), bucket.Id, key)
	if err != nil {
		_ = fs.DeleteFile(objectID)
		log.Printf("Database error reading existing object: %v\n", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	oldFiles, err := h.db.ReplaceObject(r.Context(), database.Object{
		ObjectId:           objectID,
		BucketId:           bucket.Id,
		ObjectKey:          key,
		SizeBytes:          sizeBytes,
		ContentType:        contentType,
		ContentDisposition: contentDisposition,
		ContentLanguage:    contentLanguage,
		CustomMetadata:     customMetadata,
		ETag:               etag,
	}, []database.ObjectSegment{{FileID: objectID, Size: sizeBytes, Start: 0}})
	if err != nil {
		_ = fs.DeleteFile(objectID)
		log.Printf("Database error creating object: %v\n", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if err := fs.DeleteFiles(oldFiles); err != nil {
		log.Printf("cleanup after overwrite: %v", err)
	}

	if existingObj != nil {
		if err := fs.DeleteFile(existingObj.ObjectId); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("Failed to delete overwritten file %s: %v\n", existingObj.ObjectId, err)
		}
	}

	w.Header().Set("ETag", fmt.Sprintf("\"%s\"", etag))
	w.Header().Set("X-Amz-Checksum-Crc32", serverCRC32)
	w.WriteHeader(http.StatusOK)
}
