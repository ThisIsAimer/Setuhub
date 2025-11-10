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
		myErr := utils.ErrorHandler(fmt.Errorf("no postid given"), "No postid given", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
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
		myErr := utils.ErrorHandler(err, "Error decoding json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	newUser.Uuid = strings.TrimSpace(newUser.Uuid)
	newUser.Email = strings.TrimSpace(newUser.Email)
	newUser.Password = strings.TrimSpace(newUser.Password)

	if newUser.Email == "" || newUser.Uuid == "" {
		myErr := utils.ErrorHandler(fmt.Errorf("please send all required fields"), "Please send all required fields", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	err = isValidEmailFormat(newUser.Email)

	if err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(newUser.Password) < 6 {
		myErr := utils.ErrorHandler(fmt.Errorf("password too short"), "Password must be at least 6 characters", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	if newUser.Password != newUser.ConfirmPassword {
		myErr := utils.ErrorHandler(fmt.Errorf("password invalid"), "Password invalid", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	newUser, myErr := databasehandler.SignUpDBHandler(newUser)
	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	tokenString, err := utils.SignToken(newUser.Uuid, newUser.Role, newUser.Authentication)
	if err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusInternalServerError)
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
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

}

// otp -------------------------------------------------------------------------------------------
func SignUpOtpfunc(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)

	if !ok {
		utils.WriteJSONError(w, "No user id in jwt", http.StatusUnauthorized)
		return
	}

	auth, ok := r.Context().Value(utils.JwtKey("auth")).(string)
	if !ok {
		utils.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	role, ok := r.Context().Value(utils.JwtKey("role")).(string)

	if !ok {
		utils.WriteJSONError(w, "No user id in jwt", http.StatusUnauthorized)
		return
	}

	if auth != "unverified" {

		if auth == "mail" {
			utils.WriteJSONError(w, "Mail already verified", http.StatusBadRequest)
			return
		}

		if auth == "verified" {
			utils.WriteJSONError(w, "User is authenticated", http.StatusBadRequest)
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
		myErr := utils.ErrorHandler(err, "Error decoding json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	var user models.User

	user, myErr := databasehandler.SignupOtpDBHandler(uuid, role, otp.Otp)

	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	tokenString, err := utils.SignToken(user.Uuid, user.Role, user.Authentication)
	if err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusInternalServerError)
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
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

}

// authenticate ------------------------------------------------------------------------------------------------

func AuthenticationHandler(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok {
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	auth, ok := r.Context().Value(utils.JwtKey("auth")).(string)
	if !ok {
		utils.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if auth == "verified" {
		utils.WriteJSONError(w, "User already verified", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var userInfo models.UserInfo

	err := decoder.Decode(&userInfo)
	if err != nil {
		myErr := utils.ErrorHandler(err, "Error decoding json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	err = checkEmptyField(userInfo)

	if err != nil {
		myErr := utils.ErrorHandler(err, "One or user info fields empty", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	user, myErr := databasehandler.AuthenticationDBhandler(uuid, userInfo)

	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	tokenString, err := utils.SignToken(user.Uuid, user.Role, user.Authentication)
	if err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusInternalServerError)
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
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
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
		myErr := utils.ErrorHandler(err, "Error decoding json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	if user.Email == "" || user.Password == "" {
		myErr := utils.ErrorHandler(err, "Username and password are required", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	user, myErr := databasehandler.LoginDBHandlerFunc(user.Email, user.Password)
	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	tokenString, err := utils.SignToken(user.Uuid, user.Role, user.Authentication)
	if err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusInternalServerError)
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
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
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
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
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
		myErr := utils.ErrorHandler(err, "Error decoding json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	if req.Email == "" {
		utils.WriteJSONError(w, "Please enter an email", http.StatusBadRequest)
		return
	}

	user, myErr := databasehandler.ForgotPasswordDBHandler(req.Email)
	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	tokenString, err := utils.SignToken(user.Uuid, user.Role, user.Authentication)
	if err != nil {
		utils.WriteJSONError(w, err.Error(), http.StatusInternalServerError)
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
		Message string `json:"message"`
	}{
		Status: "Success",
		Message: fmt.Sprintf("Otp sent to email : %s", req.Email),
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

}

// reset password------------------------------------------------------------------------------------------------------
func ResetPassHandler(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok {
		utils.WriteJSONError(w, "no user id in jwt", http.StatusUnauthorized)
		return
	}

	var req models.ResetPassword

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	decoder.DisallowUnknownFields()

	err := decoder.Decode(&req)

	if err != nil {
		myErr := utils.ErrorHandler(err, "Error decoding json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	req.NewPassword = strings.TrimSpace(req.NewPassword)

	if len(req.NewPassword) < 6 {
		myErr := utils.ErrorHandler(fmt.Errorf("password less then 6 characters"), "Password must have atleast 6 characters", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	if req.NewPassword == "" {
		myErr := utils.ErrorHandler(fmt.Errorf("new or confirm passwords are empty"), "Empty json fields", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		myErr := utils.ErrorHandler(fmt.Errorf("new pass doesnt match confirm pass"), "New password and confirm password do not match", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	myErr := databasehandler.ResetPassExecDBHandler(uuid, req.Otp, req.NewPassword)
	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := struct {
		Status string `json:"status"`
		Message string `json:"message"`
	}{
		Status: "Success",
		Message: "Password updated successfully, go login with the new password",
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

}

// stores location-----------------------------------------------------------------------------------------------------

func UpdateCoordinatesHandlerFunc(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok {
		utils.WriteJSONError(w, "No user id in jwt", http.StatusUnauthorized)
		return
	}

	var coordinates models.Coordinates

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	decoder.DisallowUnknownFields()

	err := decoder.Decode(&coordinates)

	if err != nil {
		myErr := utils.ErrorHandler(err, "Invalid json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	err = checkEmptyField(coordinates)

	if err != nil {
		myErr := utils.ErrorHandler(err, " One or user info fields empty", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	myErr := databasehandler.UpdateCoordinates(uuid, coordinates)
	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
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
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

}

// view profile -----------------------------------------------------------------------------------------------------------------------
func ViewProfile(w http.ResponseWriter, r *http.Request) {

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok {
		utils.WriteJSONError(w, "No user id in jwt", http.StatusUnauthorized)
		return
	}

	user, myErr := databasehandler.ProfileInfoDB(uuid)
	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Status string      `json:"status"`
		Data   models.User `json:"data"`
	}{
		Status: "Success",
		Data:   user,
	}

	err := json.NewEncoder(w).Encode(response)

	if err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

}

// update profile pic -------------------------------------------------------------------------------------
func UpdateProfilePhoto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPatch {
		utils.WriteJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uuid, ok := r.Context().Value(utils.JwtKey("uuid")).(string)
	if !ok || uuid == "" {
		utils.WriteJSONError(w, "No user id in jwt", http.StatusUnauthorized)
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
		myErr := utils.ErrorHandler(err, "Invalid json body", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	request.ProfilePhotoUrl = strings.TrimSpace(request.ProfilePhotoUrl)
	if request.ProfilePhotoUrl == "" {
		myErr := utils.ErrorHandler(fmt.Errorf("profilePhotoUrl is required"), "ProfilePhotoUrl is required", http.StatusBadRequest)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	myErr := databasehandler.UpdateProfilePhotoDB(uuid, request.ProfilePhotoUrl)

	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Status string `json:"status"`
	}{
		Status: "Success",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}
}
