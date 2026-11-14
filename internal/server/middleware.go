package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
)

var (
	allowHeaders = strings.Join([]string{
		"Accept",
		"Authorization",
		"Cache-Control",
		"Content-Disposition",
		"Content-Encoding",
		"Content-MD5",
		"Content-Type",
		"Origin",
		"Range",
		"Priority",

		// AWS Signature v4
		"X-Amz-Date",
		"X-Amz-Content-Sha256",
		"X-Amz-Security-Token",
		"X-Amz-User-Agent",

		// AWS SDK v3
		"Amz-Sdk-Invocation-Id",
		"Amz-Sdk-Request",

		// AWS checksum
		"X-Amz-Checksum-Algorithm",
		"X-Amz-Checksum-Mode",
		"X-Amz-Sdk-Checksum-Algorithm",
		"X-Amz-Checksum-Crc32",
		"X-Amz-Checksum-Crc32c",
		"X-Amz-Checksum-Sha1",
		"X-Amz-Checksum-Sha256",

		// Object operations
		"X-Amz-Copy-Source",
		"X-Amz-Acl",
		"X-Amz-Storage-Class",
		"X-Amz-Server-Side-Encryption",

		// Custom metadata
		"X-Amz-Meta-Unique-Id",

		// Conditional requests
		"If-Match",
		"If-None-Match",
		"If-Modified-Since",
		"If-Unmodified-Since",
	}, ", ")

	exposeHeaders = strings.Join([]string{
		"ETag",
		"Last-Modified",
		"Content-Length",
		"Content-Type",
		"Content-Range",
		"Content-Encoding",
		"Content-Disposition",
		"Cache-Control",
		"Expires",
		"Accept-Ranges",
		"X-Amz-Request-Id",
		"X-Amz-Version-Id",
		"X-Amz-Delete-Marker",
		"X-Amz-Meta-Unique-Id",
		"X-Amz-Checksum-Crc32",
		"X-Amz-Checksum-Crc32c",
		"X-Amz-Checksum-Sha1",
		"X-Amz-Checksum-Sha256",
	}, ", ")
)

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Advertise HTTP/3 support
		if s.http3Enabled && s.http3Server != nil && r.ProtoMajor < 3 {
			s.http3Server.SetQUICHeaders(w.Header())
		}

		// CORS headers
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
		h.Set("Access-Control-Allow-Headers", allowHeaders)
		h.Set("Access-Control-Expose-Headers", exposeHeaders)
		h.Set("Access-Control-Max-Age", "86400")

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		b := make([]byte, 8)
		rand.Read(b)
		id := fmt.Sprintf("%X", b)
		h.Set("x-amz-request-id", id)

		ctx := context.WithValue(r.Context(), "requestID", id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
