package utils

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
)

func ErrorResponse(w http.ResponseWriter, message string, status int) {
	data := map[string]string{
		"error": message,
	}

	jsonData, _ := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(jsonData); err != nil {
		log.Printf("Failed to write error response: %v", err)
	}
}

func SuccessResponse(w http.ResponseWriter, data interface{}, status int) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(jsonData)
	return err
}

type S3Error struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource"`
	RequestId string   `xml:"RequestId"`
}

var s3StatusMap = map[string]int{
	// 404 - Not Found
	"NoSuchBucket": http.StatusNotFound,
	"NoSuchKey":    http.StatusNotFound,
	"NoSuchUpload": http.StatusNotFound,

	// 403 - Forbidden
	"AccessDenied":          http.StatusForbidden,
	"SignatureDoesNotMatch": http.StatusForbidden,
	"RequestTimeTooSkewed":  http.StatusForbidden, // Clock drift > 5 mins
	"ExpiredToken":          http.StatusForbidden,
	"InvalidAccessKeyId":    http.StatusForbidden,
	"InvalidToken":          http.StatusForbidden,

	// 400 - Bad Request
	"InvalidBucketName":       http.StatusBadRequest,
	"InvalidDigest":           http.StatusBadRequest, // Checksum mismatch
	"BucketAlreadyExists":     http.StatusBadRequest, // Global name conflict
	"BucketAlreadyOwnedByYou": http.StatusBadRequest,
	"EntityTooLarge":          http.StatusBadRequest, // File exceeds max size
	"EntityTooSmall":          http.StatusBadRequest,
	"InvalidURI":              http.StatusBadRequest,
	"MetadataTooLarge":        http.StatusBadRequest,
	"KeyTooLongError":         http.StatusBadRequest,

	// 409 - Conflict
	"BucketNotEmpty": http.StatusConflict,

	// 500 - Server Errors
	"InternalError":      http.StatusInternalServerError,
	"ServiceUnavailable": http.StatusServiceUnavailable,
}

func S3ErrorResponse(w http.ResponseWriter, e S3Error) {
	statusCode, exists := s3StatusMap[e.Code]
	if !exists {
		statusCode = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(statusCode)

	if _, err := w.Write([]byte(xml.Header)); err != nil {
		log.Printf("Failed to write XML header: %v", err)
	}

	xmlData, err := xml.Marshal(e)
	if err != nil {
		log.Printf("Failed to encode S3 error: %v", err)
	} else if _, err := w.Write(xmlData); err != nil {
		log.Printf("Failed to write XML data: %v", err)
	}
}
