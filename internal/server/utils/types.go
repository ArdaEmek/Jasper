package utils

import (
	"encoding/xml"
	"s3/internal/database"
)

type CreateMultipartUploadResponse struct {
	XMLName  xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ InitiateMultipartUploadResult"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadId string   `xml:"UploadId"`
}

type CompleteMultipartUploadRequest struct {
	XMLName xml.Name `xml:"CompleteMultipartUpload"`
	Parts   []struct {
		PartNumber int    `xml:"PartNumber"`
		ETag       string `xml:"ETag"`
	} `xml:"Part"`
}
type CompleteMultipartUploadResponse struct {
	XMLName  xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CompleteMultipartUploadResult"`
	Location string   `xml:"Location"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	ETag     string   `xml:"ETag"`
}

type ListPartsResponse struct {
	XMLName              xml.Name                       `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListPartsResult"`
	Bucket               string                         `xml:"Bucket"`
	Key                  string                         `xml:"Key"`
	UploadId             string                         `xml:"UploadId"`
	PartNumberMarker     int                            `xml:"PartNumberMarker"`
	NextPartNumberMarker int                            `xml:"NextPartNumberMarker"`
	MaxParts             int                            `xml:"MaxParts"`
	IsTruncated          bool                           `xml:"IsTruncated"`
	Parts                []database.MultipartUploadPart `xml:"Part"`
}

type DeleteObjectsRequest struct {
	XMLName xml.Name       `xml:"Delete"`
	Quiet   bool           `xml:"Quiet"`
	Objects []DeleteObject `xml:"Object"`
}

type DeleteObject struct {
	Key  string `xml:"Key"`
	ETag string `xml:"ETag,omitempty"`
	Size *int64 `xml:"Size,omitempty"`
}

type DeleteObjectsResult struct {
	XMLName xml.Name        `xml:"http://s3.amazonaws.com/doc/2006-03-01/ DeleteResult"`
	Deleted []DeletedObject `xml:"Deleted,omitempty"`
	Errors  []DeleteError   `xml:"Error,omitempty"`
}

type DeletedObject struct {
	Key string `xml:"Key"`
}

type DeleteError struct {
	Key     string `xml:"Key"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}
