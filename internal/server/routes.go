package server

import (
	"net/http"
	"s3/internal/server/routes"
	"s3/internal/server/routes/admin"
)

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	// Homepage routes
	homeR := routes.New()
	homeR.RegisterEndpoints(mux)

	// Admin Routes
	adminR := admin.New()
	adminR.RegisterEndpoints(mux)

	return s.corsMiddleware(mux)
}
