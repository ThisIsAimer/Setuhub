package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"hackathon/internal/models"
	databasehandler "hackathon/internal/repositories/sqlconnect/database_handler"
	"hackathon/pkg/utils"
)

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

	if newUser.Password != newUser.ConfirmPassword {
		http.Error(w, utils.ErrorHandler(fmt.Errorf("passwords dont match"), "passwords dont match").Error(), http.StatusBadRequest)
		return
	}

	newUser, err = databasehandler.SignUpDBHandler(newUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	var aUser models.User

	err := decoder.Decode(&aUser)
	if err != nil {
		http.Error(w, utils.ErrorHandler(err, "error decoding json body").Error(), http.StatusBadRequest)
		return
	}

	if aUser.Email == "" || aUser.Password == "" {
		myErr := utils.ErrorHandler(err, "username and password are required")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	givenPass := aUser.Password

	aUser, err = databasehandler.LoginDBHandlerFunc(givenPass, aUser.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = utils.VerifyPassword(givenPass, aUser.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tokenString, err := utils.SignToken(aUser.Uuid, aUser.Role, aUser.Authentication)
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

	responce := struct {
		Status string `json:"Status"`
	}{
		Status: "success",
	}

	err = json.NewEncoder(w).Encode(responce)

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

	w.Write([]byte("Message: logged out successfully"))
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

	err = databasehandler.ForgotPasswordDBHandler(req.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	responce := struct {
		Status string `json:"status"`
	}{
		Status: fmt.Sprintf("Sent reset link to email : %s", req.Email),
	}

	err = json.NewEncoder(w).Encode(responce)

	if err != nil {
		myErr := utils.ErrorHandler(err, "error encoding json")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

}

// reset password------------------------------------------------------------------------------------------------------
func ResetPassHandler(w http.ResponseWriter, r *http.Request) {

	token := r.PathValue("resetcode")

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
	if req.ConfirmPassword == "" || req.NewPassword == "" {
		myErr := utils.ErrorHandler(fmt.Errorf("new or confirm passwords are empty"), "empty json fields")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		myErr := utils.ErrorHandler(fmt.Errorf("new pass doesnt match confirm pass"), "both password fields doesnt match")
		http.Error(w, myErr.Error(), http.StatusBadRequest)
		return
	}

	err = databasehandler.ResetPassExecDBHandler(token, req.NewPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	responce := struct {
		Status string `json:"message"`
	}{
		Status: "password updated successfully",
	}

	err = json.NewEncoder(w).Encode(responce)

	if err != nil {
		myErr := utils.ErrorHandler(err, "error encoding json")
		http.Error(w, myErr.Error(), http.StatusInternalServerError)
		return
	}

}
