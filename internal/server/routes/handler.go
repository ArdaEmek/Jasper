package routes

import (
	"net/http"
	"s3/internal/database"
	"s3/internal/server/utils"
)

type Handler struct {
	db database.Service
}

func New() *Handler {
	db := database.New()
	return &Handler{db: db}
}

func (h *Handler) RegisterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("GET /", homepageGET)
}

func homepageGET(w http.ResponseWriter, r *http.Request) {
	res := map[string]string{"message": "Hello World"}
	utils.SuccessResponse(w, res, 200)
}
