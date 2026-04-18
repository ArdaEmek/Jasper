package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Advertise HTTP/3 support
		if r.ProtoMajor < 3 {
			s.http3Server.SetQUICHeaders(w.Header())
		}

		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Amz-Date, X-Amz-Content-Sha256, x-amz-meta-unique-id")
		w.Header().Set("Access-Control-Expose-Headers", "ETag, x-amz-request-id")
		w.Header().Set("Access-Control-Allow-Credentials", "false")

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		b := make([]byte, 8)
		rand.Read(b)
		id := fmt.Sprintf("%X", b)

		w.Header().Set("x-amz-request-id", id)

		ctx := context.WithValue(r.Context(), "requestID", id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
