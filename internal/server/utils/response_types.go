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
