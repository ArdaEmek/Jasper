package admin

import (
	"encoding/json"
	"log"
	"net/http"
	"s3/internal/database"
	"s3/internal/server/utils"
	"strconv"
)

type jsonData struct {
	Id int `json:"id"`
}

func (h *Handler) userGET(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var data jsonData
	err := decoder.Decode(&data)
	if err != nil {
		log.Printf("Error occurred while reading JSON data: %v", err)

		utils.ErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	res, err := h.db.GetUser(r.Context(), data.Id)
	if err != nil {
		utils.ErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if res == nil {
		utils.ErrorResponse(w, "User not found", http.StatusNotFound)
		return
	}

	utils.SuccessResponse(w, res, 200)
}

func (h *Handler) userPOST(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var data map[string]string
	err := decoder.Decode(&data)
	if err != nil {
		log.Printf("Error occurred while reading JSON data: %v", err)

		utils.ErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	getField := func(field string) (string, bool) {
		value, ok := data[field]
		if !ok {
			utils.ErrorResponse(w, "Missing fields", http.StatusBadRequest)
		}
		return value, ok
	}

	username, ok := getField("username")
	if !ok {
		return
	}

	email, ok := getField("email")
	if !ok {
		return
	}

	tempuser := database.RegisterParams{
		Name:  username,
		Email: email,
	}

	user, err := h.db.CreateUser(r.Context(), tempuser)
	if err != nil {
		utils.ErrorResponse(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	res := map[string]string{
		"message": "Created user successfully",
		"id":      strconv.Itoa(user.Id),
	}

	utils.SuccessResponse(w, res, 200)
}
