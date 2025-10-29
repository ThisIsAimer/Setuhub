package handlers

import (
	"encoding/json"
	"hackathon/internal/models"
	databasehandler "hackathon/internal/repositories/sqlconnect/database_handler"
	"hackathon/pkg/utils"
	"net/http"
)

func HelpRequestPost(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var newPost models.Post

	err := decoder.Decode(&newPost)
	if err != nil {
		http.Error(w, utils.ErrorHandler(err, "error decoding json body").Error(), http.StatusBadRequest)
		return
	}

	err = databasehandler.HelpRequestPostDB(uuid, newPost)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status string `json:"status"`
	}{
		Status: "Success",
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

}

func HelpRequestGet(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var coordinates models.Coordinates

	err := decoder.Decode(&coordinates)
	if err != nil {
		http.Error(w, utils.ErrorHandler(err, "error decoding json body").Error(), http.StatusBadRequest)
		return
	}

	posts, err := databasehandler.HelpRequestGetDB(coordinates)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var status string

	if len(posts) == 0 {
		status = "no posts available"
	} else {
		status = "success"
	}

	response := struct {
		Status string        `json:"status"`
		Data   []models.Post `json:"data"`
	}{
		Status: status,
		Data:   posts,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}
}
