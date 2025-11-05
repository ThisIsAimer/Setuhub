package handlers

import (
	"encoding/json"
	"fmt"
	"hackathon/internal/models"
	databasehandler "hackathon/internal/repositories/sqlconnect/database_handler"
	"hackathon/pkg/utils"
	"net/http"
	"strconv"
	"strings"
)

func HandleRequestCreate(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	section := r.PathValue("section")

	section = strings.TrimSpace(section)

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

	if section != "moments" {
		if newPost.Longitude == 0 && newPost.Latitude == 0 {
			http.Error(w, utils.ErrorHandler(fmt.Errorf("invalid coordinates: pointing to null island"), "no coordinates provided").Error(), http.StatusBadRequest)
			return
		}
	}

	newPost.Title = strings.TrimSpace(newPost.Title)
	newPost.Description = strings.TrimSpace(newPost.Description)

	if newPost.Description == "" {
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

	if section != "moments" {
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
	} else {

		response := struct {
			Status             string      `json:"status"`
			Data               models.Post `json:"data"`
		}{
			Status:             "Success",
			Data:               newPost,
		}

		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			myErr := utils.ErrorHandler(err, "Failed to encode response")
			http.Error(w, myErr.Error(), http.StatusInternalServerError)
			return
		}

	}

}

func HandleRequestRetrieve(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	section := r.PathValue("section")

	section = strings.TrimSpace(section)

	_, err := checkSection(section)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var coordinates models.Coordinates

	if section != "moments" {
		query := r.URL.Query()

		latStr := query.Get("latitude")
		lngStr := query.Get("longitude")

		if strings.TrimSpace(latStr) == "" || strings.TrimSpace(lngStr) == "" {
			myErr := utils.ErrorHandler(fmt.Errorf("lat or long not given"), "lat or long not given")
			http.Error(w, myErr.Error(), http.StatusInternalServerError)
			return
		}

		coordinates.Latitude, err = strconv.ParseFloat(latStr, 64)
		if err != nil {
			http.Error(w, utils.ErrorHandler(err, "invalid latitude parameter").Error(), http.StatusBadRequest)
			return
		}

		coordinates.Longitude, err = strconv.ParseFloat(lngStr, 64)
		if err != nil {
			http.Error(w, utils.ErrorHandler(err, "invalid longitude parameter").Error(), http.StatusBadRequest)
			return
		}
	}

	posts, err := databasehandler.RetrieveRequestGetDB(uuid, section, coordinates)

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

// my Requests-------------------------------------------------------------------------------------------------------
func HandleMyRequestRetrieve(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	section := r.PathValue("section")

	section = strings.TrimSpace(section)

	_, err := checkSection(section)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	posts, err := databasehandler.RetrieveMyRequestGetDB(uuid, section)

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

func InterestedPostHandler(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	postUuid := r.PathValue("postuuid")

	postUuid = strings.TrimSpace(postUuid)

	if postUuid == "" {
		http.Error(w, "invalid postuuid", http.StatusUnauthorized)
		return
	}

	result, err := databasehandler.InterestedPostHandler(uuid, postUuid)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status          string `json:"status"`
		Change          bool   `json:"change"`
		InterestedCount int    `json:"interestedCount"`
	}{
		Status:          "success",
		Change:          result.Changed,
		InterestedCount: result.InterestedCount,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

}

func UninterestedPostHandler(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	postUuid := strings.TrimSpace(r.PathValue("postuuid"))
	if postUuid == "" {
		http.Error(w, "invalid postuuid", http.StatusBadRequest)
		return
	}

	result, err := databasehandler.UninterestedPost(uuid, postUuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Status          string `json:"status"`
		Change          bool   `json:"change"`
		InterestedCount int    `json:"interestedCount"`
	}{
		Status:          "success",
		Change:          result.Changed,
		InterestedCount: result.InterestedCount,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}
}
