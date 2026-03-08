package server

import (
	"net/http"
	"s3/internal/server/handlers"
)

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	// Creating route handler
	h := handlers.New(s.db)
	h.RegisterEndpoints(mux)

	// Wrap the mux with CORS middleware
	return s.corsMiddleware(mux)
}
