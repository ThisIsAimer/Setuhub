package handlers

import (
	"encoding/json"
	"fmt"
	"hackathon/internal/models"
	databasehandler "hackathon/internal/repositories/sqlconnect/database_handler"
	"hackathon/pkg/utils"
	"net/http"
	"strings"
)

func HandleRequestCreate(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	section := r.PathValue("section")

	postType, err := checkSection(section)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var newPost models.Post

	err = decoder.Decode(&newPost)
	if err != nil {
		http.Error(w, utils.ErrorHandler(err, "error decoding json body").Error(), http.StatusBadRequest)
		return
	}

	newPost.Type = postType

	if newPost.Longitude == 0 && newPost.Latitude == 0 {
		http.Error(w, utils.ErrorHandler(fmt.Errorf("invalid coordinates: pointing to null island"), "no coordinates provided").Error(), http.StatusBadRequest)
		return
	}

	newPost.Title = strings.TrimSpace(newPost.Title)
	newPost.Description = strings.TrimSpace(newPost.Description)

	if newPost.Description == ""{
		http.Error(w, utils.ErrorHandler(fmt.Errorf("no description Provided"), "no description Provided").Error(), http.StatusBadRequest)
		return
	}

	

	noti := make(chan string)

	newPost, err = databasehandler.CreateRequestPostDB(uuid, section, newPost, noti)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status             string      `json:"status"`
		Data               models.Post `json:"data"`
		NotificationStatus string      `json:"notificationStatus"`
	}{
		Status:             "Success",
		Data:               newPost,
		NotificationStatus: <-noti,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

}

func HandleRequestRetrieve(w http.ResponseWriter, r *http.Request) {

	section := r.PathValue("section")

	_, err := checkSection(section)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var coordinates models.Coordinates

	err = decoder.Decode(&coordinates)
	if err != nil {
		http.Error(w, utils.ErrorHandler(err, "error decoding json body").Error(), http.StatusBadRequest)
		return
	}

	posts, err := databasehandler.RetrieveRequestGetDB(section, coordinates)

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

func HandleRequestDone(w http.ResponseWriter, r *http.Request) {
	postId := r.PathValue("postid")

	if postId == "" {
		myErr := utils.ErrorHandler(fmt.Errorf("no postid given"), "no postid given")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

	err := databasehandler.DonePatchRequestDB(postId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status string `json:"status"`
	}{
		Status: "success",
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}
}
