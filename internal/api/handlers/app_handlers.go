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
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	section := r.PathValue("section")

	section = strings.TrimSpace(section)

	postType, err := checkSection(section)

	if err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var newPost models.Post

	err = decoder.Decode(&newPost)
	if err != nil {
		myErr := utils.ErrorHandler(err, "error decoding json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	newPost.Type = postType

	if section != "moments" {
		if newPost.Longitude == 0 && newPost.Latitude == 0 {
			myErr := utils.ErrorHandler(fmt.Errorf("invalid coordinates: pointing to null island"), "no coordinates provided", http.StatusBadRequest)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}
	}

	newPost.Title = strings.TrimSpace(newPost.Title)
	newPost.Description = strings.TrimSpace(newPost.Description)

	if newPost.Description == "" {
		myErr := utils.ErrorHandler(fmt.Errorf("no description Provided"), "no description Provided", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	noti := make(chan string)

	newPost, myErr := databasehandler.CreateRequestPostDB(uuid, section, newPost, noti)

	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
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
			myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}
	} else {

		response := struct {
			Status string      `json:"status"`
			Data   models.Post `json:"data"`
		}{
			Status: "Success",
			Data:   newPost,
		}

		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}

	}

}

func HandleRequestRetrieve(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	section := r.PathValue("section")

	section = strings.TrimSpace(section)

	_, err := checkSection(section)
	if err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var coordinates models.Coordinates

	query := r.URL.Query()

	var page int

	pageStr := strings.TrimSpace(query.Get("page"))

	if pageStr == "" {

		page = 1

	} else {
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			myErr := utils.ErrorHandler(err, "invalid page", http.StatusBadRequest)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}

		if page <= 0 {
			page = 1
		}

	}

	if section != "moments" {

		latStr := query.Get("latitude")
		lngStr := query.Get("longitude")

		if strings.TrimSpace(latStr) == "" || strings.TrimSpace(lngStr) == "" {
			myErr := utils.ErrorHandler(fmt.Errorf("lat or long not given"), "lat or long not given", http.StatusBadRequest)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}

		coordinates.Latitude, err = strconv.ParseFloat(latStr, 64)
		if err != nil {
			myErr := utils.ErrorHandler(err, "invalid latitude parameter", http.StatusBadRequest)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}

		coordinates.Longitude, err = strconv.ParseFloat(lngStr, 64)
		if err != nil {
			myErr := utils.ErrorHandler(err, "invalid longitude parameter", http.StatusBadRequest)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}
	}

	posts, myErr := databasehandler.RetrieveRequestGetDB(uuid, section, page, coordinates)

	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
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
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}
}

// my Requests-------------------------------------------------------------------------------------------------------
func HandleMyRequestRetrieve(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}
	query := r.URL.Query()

	var page int

	pageStr := strings.TrimSpace(query.Get("page"))

	if pageStr == "" {
		page = 1
	} else {
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			myErr := utils.ErrorHandler(err, "invalid page", http.StatusBadRequest)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}

		if page <= 0 {
			page = 1
		}

	}

	section := r.PathValue("section")

	section = strings.TrimSpace(section)

	_, err := checkSection(section)
	if err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	posts, myErr := databasehandler.RetrieveMyRequestGetDB(uuid, section, page)

	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
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
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}
}

// expo token-----------------------------------------------------------------------------------------------------------------
func SetFirebaseToken(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request = struct {
		FirebaseToken string `json:"firebaseToken,omitempty"`
	}{}

	err := decoder.Decode(&request)
	if err != nil {
		myErr := utils.ErrorHandler(err, "error decoding json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return

	}

	myErr := databasehandler.SetFirebaseTokenDbHandler(uuid, request.FirebaseToken)

	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	response := struct {
		Status string `json:"status"`
	}{
		Status: "success",
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}
}

// done -----------------------------------------------------------------------------------------------------
func HandleRequestDone(w http.ResponseWriter, r *http.Request) {
	postId := r.PathValue("postid")

	if postId == "" {
		myErr := utils.ErrorHandler(fmt.Errorf("no postid given"), "no postid given", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	myErr := databasehandler.DonePatchRequestDB(postId)
	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status string `json:"status"`
	}{
		Status: "success",
	}

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}
}

func InterestedPostHandler(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	postUuid := r.PathValue("postuuid")

	postUuid = strings.TrimSpace(postUuid)

	if postUuid == "" {
		utils.WriteJSONError(w, "invalid postuuid", http.StatusUnauthorized)
		return
	}

	result, myErr := databasehandler.InterestedPostHandler(uuid, postUuid)

	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
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

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

}

func UninterestedPostHandler(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok {
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	postUuid := strings.TrimSpace(r.PathValue("postuuid"))
	if postUuid == "" {
		utils.WriteJSONError(w, "invalid postuuid", http.StatusBadRequest)
		return
	}

	result, myErr := databasehandler.UninterestedPost(uuid, postUuid)
	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
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
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}
}

// comment handlers--------------------------------------------------------------------------------------------------------

func GetCommentHandler(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query()

	var page int

	pageStr := strings.TrimSpace(query.Get("page"))

	if pageStr == "" {
		page = 1
	} else {
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			myErr := utils.ErrorHandler(err, "invalid page", http.StatusBadRequest)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}

		if page <= 0 {
			page = 1
		}

	}

	postUuid := r.PathValue("postuuid")

	postUuid = strings.TrimSpace(postUuid)

	if postUuid == "" {
		utils.WriteJSONError(w, "invalid commentUuid", http.StatusBadRequest)
		return
	}

	comments, myErr := databasehandler.GetCommentDBHandler(postUuid, uuid, page)
	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Status   string           `json:"status"`
		Comments []models.Comment `json:"comments"`
	}{
		Status:   "success",
		Comments: comments,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

}

func CreateCommentHandler(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	var comment models.Comment

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&comment)
	if err != nil {
		myErr := utils.ErrorHandler(err, "error decoding json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	comment.Content = strings.TrimSpace(comment.Content)
	if comment.Content == "" {
		utils.WriteJSONError(w, "comment empty", http.StatusBadRequest)
		return
	}

	comment.Uuid = uuid

	comment, myErr := databasehandler.CreateCommentDBHandler(comment)

	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Status  string         `json:"status"`
		Comment models.Comment `json:"comment"`
	}{
		Status:  "success",
		Comment: comment,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

}

func EditCommentHandler(w http.ResponseWriter, r *http.Request) {
	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	commentUuid := r.PathValue("commentuuid")

	commentUuid = strings.TrimSpace(commentUuid)

	if commentUuid == "" {
		utils.WriteJSONError(w, "invalid commentUuid", http.StatusBadRequest)
		return
	}

	var request = struct {
		Content string `json:"content"`
	}{}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&request)
	if err != nil {
		myErr := utils.ErrorHandler(err, "error decoding json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	request.Content = strings.TrimSpace(request.Content)
	if request.Content == "" {
		utils.WriteJSONError(w, "content empty", http.StatusBadRequest)
		return
	}

	edited, myErr := databasehandler.EditCommentDBHandler(commentUuid, uuid, request.Content)
	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Status string `json:"status"`
		Edited bool   `json:"edited"`
	}{
		Status: "success",
		Edited: edited,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}
}

func DeleteCommentHandler(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	commentUuid := r.PathValue("commentuuid")

	commentUuid = strings.TrimSpace(commentUuid)

	if commentUuid == "" {
		utils.WriteJSONError(w, "invalid commentUuid", http.StatusBadRequest)
		return
	}

	deleted, count, myErr := databasehandler.DeleteCommentDBHandler(commentUuid, uuid)

	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status       string `json:"status"`
		Deleted      bool   `json:"deleted"`
		CommentCount int    `json:"commentCount"`
	}{
		Status:       "success",
		Deleted:      deleted,
		CommentCount: count,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

}
