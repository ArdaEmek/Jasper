package admin

import (
	"net/http"
	"os"
	"s3/internal/database"
)

var (
	adminEndpointEnabled = os.Getenv("ADMIN_ENDPOINT_ENABLED")
	adminEndpointKey     = os.Getenv("ADMIN_ENDPOINT_KEY")
)

type Handler struct {
	db database.Service
}

func New() *Handler {
	db := database.New()
	return &Handler{db: db}
}

func (h *Handler) RegisterEndpoints(mux *http.ServeMux) {
	if adminEndpointEnabled != "true" {
		return
	}

	mux.HandleFunc("GET /admin/user", h.withAuth(h.userGET))
	mux.HandleFunc("POST /admin/user", h.withAuth(h.userPOST))
}

func (h *Handler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate token
		if token != adminEndpointKey {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
