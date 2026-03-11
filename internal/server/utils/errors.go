package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

func ErrorResponse(w http.ResponseWriter, message string, status int) {
	data := map[string]string{
		"error": message,
	}

	jsonData, _ := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(jsonData); err != nil {
		log.Printf("Failed to write error response: %v", err)
	}
}

func SuccessResponse(w http.ResponseWriter, data interface{}, status int) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(jsonData)
	return err
}
