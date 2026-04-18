package admin

import (
	"encoding/json"
	"log"
	"net/http"
	"s3/internal/server/utils"
)

type apiKeyGetRequest struct {
	Id int `json:"id"`
}

type postApiKeyData struct {
	UserId          int    `json:"user_id"`
	PermissionLevel string `json:"permission_level"`
}

func (h *Handler) apikeyGET(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var data apiKeyGetRequest
	err := decoder.Decode(&data)
	if err != nil {
		log.Printf("Error occurred while reading JSON data: %v", err)

		utils.ErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	keys, err := h.db.GetApiKeysByUserId(r.Context(), data.Id)
	if err != nil {
		utils.ErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	utils.SuccessResponse(w, keys, 200)
}

func (h *Handler) apikeyPOST(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var data postApiKeyData
	err := decoder.Decode(&data)
	if err != nil {
		log.Printf("Error occurred while reading JSON data: %v", err)

		utils.ErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if data.UserId == 0 || data.PermissionLevel == "" {
		utils.ErrorResponse(w, "Missing fields", http.StatusBadRequest)
		return
	}

	apiKey, err := h.db.CreateApiKey(r.Context(), data.UserId, data.PermissionLevel)
	if err != nil {
		log.Printf("Error occurred while creating api key: %v", err)
		utils.ErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	utils.SuccessResponse(w, apiKey, 201)
}
