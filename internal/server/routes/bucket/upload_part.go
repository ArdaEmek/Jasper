package bucket

import (
	"bytes"
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
	"strconv"

	"github.com/klauspost/crc32"
	"github.com/segmentio/ksuid"
)

func (h *Handler) uploadPartHandler(w http.ResponseWriter, r *http.Request) {
	reqID, _ := r.Context().Value("requestID").(string)
	bucket, _ := r.Context().Value("bucket").(*database.Bucket)
	user, _ := r.Context().Value("user").(*database.User)

	key := r.PathValue("key")
	query := r.URL.Query()
	uploadID := query.Get("uploadId")
	partNumberStr := query.Get("partNumber")

	partNumber, err := strconv.Atoi(partNumberStr)
	if err != nil || partNumber < 1 || partNumber > 10000 {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InvalidArgument",
			Message:   "Part number must be an integer between 1 and 10000, inclusive",
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

	partID := ksuid.New().String()
	fw, err := fs.WriteFile(partID)
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
			_ = fs.DeleteFile(partID)
			utils.S3ErrorResponse(w, utils.S3Error{
				Code:      "InvalidDigest",
				Message:   "The Content-MD5 or checksum value that you specified is not valid.",
				RequestId: reqID,
				Resource:  r.URL.Path,
			})
			return
		}
		if !bytes.Equal(clientMD5, hash.Sum(nil)) {
			_ = fs.DeleteFile(partID)
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
		_ = fs.DeleteFile(partID)
		log.Printf("CRC32 mismatch: expected %s, got %s", clientCRC32, serverCRC32)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "BadDigest",
			Message:   "The CRC32 checksum you specified did not match the calculated checksum.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	oldFileID, err := h.db.SaveMultipartUploadPart(r.Context(), uploadID, partNumber, etag, sizeBytes, partID)
	if err != nil {
		_ = fs.DeleteFile(partID)
		log.Printf("Database error saving part metadata: %v\n", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "An internal error occurred. Try again.",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	if oldFileID != nil {
		_ = fs.DeleteFile(*oldFileID)
	}

	w.Header().Set("ETag", fmt.Sprintf("\"%s\"", etag))
	w.Header().Set("X-Amz-Checksum-Crc32", serverCRC32)
	w.WriteHeader(http.StatusOK)
}
