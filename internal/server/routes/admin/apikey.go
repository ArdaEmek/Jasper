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

type apiKeyResponse struct {
	Id              int    `json:"id"`
	UserId          int    `json:"user_id"`
	AccessKey       string `json:"access_key"`
	SecretKey       string `json:"secret_key"`
	PermissionLevel string `json:"permission_level"`
	CreatedAt       string `json:"created_at"`
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

	var resKeys []apiKeyResponse
	for _, key := range keys {
		resKeys = append(resKeys, apiKeyResponse{
			Id:              key.Id,
			UserId:          key.UserId,
			AccessKey:       key.AccessKey,
			SecretKey:       key.SecretKey,
			PermissionLevel: key.PermissionLevel,
			CreatedAt:       key.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	utils.SuccessResponse(w, resKeys, 200)
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

	res := apiKeyResponse{
		Id:              apiKey.Id,
		UserId:          apiKey.UserId,
		AccessKey:       apiKey.AccessKey,
		SecretKey:       apiKey.SecretKey,
		PermissionLevel: apiKey.PermissionLevel,
		CreatedAt:       apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	utils.SuccessResponse(w, res, 201)
}
