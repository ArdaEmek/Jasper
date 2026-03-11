package admin

import (
	"net/http"
	"s3/internal/database"
)

type Handler struct {
	db database.Service
}

func New() *Handler {
	db := database.New()
	return &Handler{db: db}
}

func (h *Handler) RegisterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/user", h.withAuth(h.userGET))
	mux.HandleFunc("POST /admin/user", h.withAuth(h.userPOST))
}

func (h *Handler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check auth token
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate token (from your db)
		if !h.db.ValidateToken(token) {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
