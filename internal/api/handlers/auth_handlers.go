package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"hackathon/internal/models"
	databasehandler "hackathon/internal/repositories/sqlconnect/database_handler"
	"hackathon/pkg/utils"
)

// home for test ---------------------------------------------------------------------------------------------------------

func Home(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("home!"))
	if err != nil {
		myErr := utils.ErrorHandler(err, "couldnt write")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}
}

//Sign in ----------------------------------------------------------------------------------------------------------------------------

func SignUpHandlerfunc(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var newUser models.User

	err := decoder.Decode(&newUser)
	if err != nil {
		http.Error(w, utils.ErrorHandler(err, "error decoding json body").Error(), http.StatusBadRequest)
		return
	}

	newUser.Uuid = strings.TrimSpace(newUser.Uuid)
	newUser.Email = strings.TrimSpace(newUser.Email)
	newUser.Password = strings.TrimSpace(newUser.Password)

	if newUser.Email == "" || newUser.Uuid == "" {
		http.Error(w, utils.ErrorHandler(fmt.Errorf("please send all required fields"), "please send all required fields").Error(), http.StatusBadRequest)
		return
	}

	err = isValidEmailFormat(newUser.Email)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(newUser.Password) < 6 {
		http.Error(w, utils.ErrorHandler(fmt.Errorf("password too short"), "password must be at least 6 characters").Error(), http.StatusBadRequest)
		return
	}

	if newUser.Password != newUser.ConfirmPassword {
		http.Error(w, utils.ErrorHandler(fmt.Errorf("passwords dont match"), "passwords dont match").Error(), http.StatusBadRequest)
		return
	}

	newUser, err = databasehandler.SignUpDBHandler(newUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tokenString, err := utils.SignToken(newUser.Uuid, newUser.Role, newUser.Authentication)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// send token as response or a cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().AddDate(0, 6, 0),
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status string `json:"status"`
		Id     string `json:"id"`
	}{
		Status: "Success",
		Id:     newUser.Uuid,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

}

// otp -------------------------------------------------------------------------------------------
func SignUpOtpfunc(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	auth, ok := r.Context().Value(utils.JwtKey("auth")).(string)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	role, ok := r.Context().Value(utils.JwtKey("role")).(string)

	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	if auth != "unverified" {

		if auth == "mail" {
			http.Error(w, "mail already verified", http.StatusBadRequest)
			return
		}

		if auth == "verified" {
			http.Error(w, "user is authenticated", http.StatusBadRequest)
			return
		}

	}

	otp := struct {
		Otp string `json:"otp" db:"otp"`
	}{}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&otp)
	if err != nil {
		http.Error(w, utils.ErrorHandler(err, "error decoding json body").Error(), http.StatusBadRequest)
		return
	}

	var user models.User

	user, err = databasehandler.SignupOtpDBHandler(uuid, role, otp.Otp)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tokenString, err := utils.SignToken(user.Uuid, user.Role, user.Authentication)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// send token as response or a cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().AddDate(0, 6, 0),
		SameSite: http.SameSiteStrictMode,
	})

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

// authenticate ------------------------------------------------------------------------------------------------

func AuthenticationHandler(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	auth, ok := r.Context().Value(utils.JwtKey("auth")).(string)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if auth == "verified" {
		http.Error(w, "User already verified", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var userInfo models.UserInfo

	err := decoder.Decode(&userInfo)
	if err != nil {
		http.Error(w, utils.ErrorHandler(err, "error decoding json body").Error(), http.StatusBadRequest)
		return
	}

	err = checkEmptyField(userInfo)

	if err != nil {
		http.Error(w, utils.ErrorHandler(err, " one or user info fields empty").Error(), http.StatusBadRequest)
		return
	}

	user, err := databasehandler.AuthenticationDBhandler(uuid, userInfo)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tokenString, err := utils.SignToken(user.Uuid, user.Role, user.Authentication)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// send token as response or a cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().AddDate(0, 6, 0),
		SameSite: http.SameSiteStrictMode,
	})

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

// login-------------------------------------------------------------------------------------------

func LoginHandlerFunc(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var user models.User

	err := decoder.Decode(&user)
	if err != nil {
		http.Error(w, utils.ErrorHandler(err, "error decoding json body").Error(), http.StatusBadRequest)
		return
	}

	if user.Email == "" || user.Password == "" {
		myErr := utils.ErrorHandler(err, "username and password are required")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	user, err = databasehandler.LoginDBHandlerFunc(user.Email, user.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tokenString, err := utils.SignToken(user.Uuid, user.Role, user.Authentication)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// send token as response or a cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().AddDate(0, 6, 0),
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status string `json:"status"`
		Id     string `json:"id"`
	}{
		Status: "Success",
		Id:     user.Uuid,
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		myErr := utils.ErrorHandler(err, "error encoding json")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

}

// log out-------------------------------------------------------------------------------------------
func LogoutHandler(w http.ResponseWriter, r *http.Request) {

	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Unix(0, 0),
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")

	responce := struct {
		Status string `json:"status"`
	}{
		Status: "logged out successfully",
	}

	err := json.NewEncoder(w).Encode(responce)

	if err != nil {
		myErr := utils.ErrorHandler(err, "error encoding json")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}
}

// forget Password----------------------------------------------------------------------------------------------
func ForgotPassHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	decoder.DisallowUnknownFields()

	err := decoder.Decode(&req)
	if err != nil {
		myErr := utils.ErrorHandler(err, "invalid json body")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "please enter an email", http.StatusBadRequest)
		return
	}

	user, err := databasehandler.ForgotPasswordDBHandler(req.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tokenString, err := utils.SignToken(user.Uuid, user.Role, user.Authentication)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// send token as response or a cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().AddDate(0, 6, 0),
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status string `json:"status"`
	}{
		Status: fmt.Sprintf("otp sent to email : %s", req.Email),
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		myErr := utils.ErrorHandler(err, "error encoding json")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

}

// reset password------------------------------------------------------------------------------------------------------
func ResetPassHandler(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	var req models.ResetPassword

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	decoder.DisallowUnknownFields()

	err := decoder.Decode(&req)

	if err != nil {
		myErr := utils.ErrorHandler(err, "invalid json body")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	req.NewPassword = strings.TrimSpace(req.NewPassword)

	if len(req.NewPassword) < 6 {
		myErr := utils.ErrorHandler(fmt.Errorf("password less then 6 characters"), "password must have atleast 6 characters")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	if req.NewPassword == "" {
		myErr := utils.ErrorHandler(fmt.Errorf("new or confirm passwords are empty"), "empty json fields")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		myErr := utils.ErrorHandler(fmt.Errorf("new pass doesnt match confirm pass"), "both password fields doesnt match")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	err = databasehandler.ResetPassExecDBHandler(uuid, req.Otp, req.NewPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status string `json:"status"`
	}{
		Status: "password updated successfully, go login with the new password",
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		myErr := utils.ErrorHandler(err, "error encoding json")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

}

// stores location-----------------------------------------------------------------------------------------------------

func UpdateCoordinatesHandlerFunc(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	var coordinates models.Coordinates

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	decoder.DisallowUnknownFields()

	err := decoder.Decode(&coordinates)

	if err != nil {
		myErr := utils.ErrorHandler(err, "invalid json body")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	err = checkEmptyField(coordinates)

	if err != nil {
		http.Error(w, utils.ErrorHandler(err, " one or user info fields empty").Error(), http.StatusBadRequest)
		return
	}

	err = databasehandler.UpdateCoordinates(uuid, coordinates)
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
		myErr := utils.ErrorHandler(err, "error encoding json")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

}

// view profile -----------------------------------------------------------------------------------------------------------------------
func ViewProfile(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	user, err := databasehandler.ProfileInfoDB(uuid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Status string      `json:"status"`
		Data   models.User `json:"data"`
	}{
		Status: "success",
		Data:   user,
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		myErr := utils.ErrorHandler(err, "error encoding json")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

}

// update profile pic -------------------------------------------------------------------------------------
func UpdateProfilePhoto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok || uuid == "" {
		http.Error(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	var request struct {
		ProfilePhotoUrl string `json:"profilePhotoUrl"`
	}

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&request)
	if err != nil {
		myErr := utils.ErrorHandler(err, "invalid json body")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	request.ProfilePhotoUrl = strings.TrimSpace(request.ProfilePhotoUrl)
	if request.ProfilePhotoUrl == "" {
		myErr := utils.ErrorHandler(fmt.Errorf("profilePhotoUrl is required"), "profilePhotoUrl is required")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	err = databasehandler.UpdateProfilePhotoDB(uuid, request.ProfilePhotoUrl)

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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		myErr := utils.ErrorHandler(err, "error encoding json")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}
}
