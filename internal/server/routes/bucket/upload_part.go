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
	"strconv"

	"github.com/klauspost/crc32"
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

	partFileName := fmt.Sprintf("%s_%d", uploadID, partNumber)
	fw, err := fs.WriteFile(partFileName)
	if err != nil {
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "Failed to prepare file upload",
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
		_ = fs.DeleteFile(partFileName)
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
		_ = fs.DeleteFile(partFileName)
		log.Printf("CRC32 mismatch: expected %s, got %s", clientCRC32, serverCRC32)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InvalidDigest",
			Message:   "The CRC32 checksum of the object does not match",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	err = h.db.SaveMultipartUploadPart(r.Context(), uploadID, partNumber, etag, sizeBytes)
	if err != nil {
		_ = fs.DeleteFile(partFileName)
		log.Printf("Database error saving part metadata: %v\n", err)
		utils.S3ErrorResponse(w, utils.S3Error{
			Code:      "InternalError",
			Message:   "Failed to save part metadata",
			RequestId: reqID,
			Resource:  r.URL.Path,
		})
		return
	}

	w.Header().Set("ETag", fmt.Sprintf("\"%s\"", etag))
	w.Header().Set("X-Amz-Checksum-Crc32", serverCRC32)
	w.WriteHeader(http.StatusOK)
}
