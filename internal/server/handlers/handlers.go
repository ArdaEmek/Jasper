package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"s3/internal/database"
)

type Handler struct {
	db database.Service
}

func New(db database.Service) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.homepageGET)
	mux.HandleFunc("GET /health", h.healthGET)
}

func (h *Handler) homepageGET(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"message": "Hello World"}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(jsonResp); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
