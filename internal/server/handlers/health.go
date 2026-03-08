package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func (h *Handler) healthGET(w http.ResponseWriter, r *http.Request) {
	resp, err := json.Marshal(h.db.Health())
	if err != nil {
		http.Error(w, "Failed to marshal health check response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(resp); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
