package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hackathon/internal/models"
	databasehandler "hackathon/internal/repositories/sqlconnect/database_handler"
	"hackathon/pkg/utils"
	"io"
	"net/http"
	"os"
	"time"
)

func CreateVerification(w http.ResponseWriter, r *http.Request) {
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

		if auth != "mail_verified" {
			utils.WriteJSONError(w, "User isnt allowed to use this", http.StatusBadRequest)
			return
		}

	if uuid == "" {
		myErr := utils.ErrorHandler(
			fmt.Errorf("user_id is required"),
			"User ID is required",
			http.StatusBadRequest,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	apiKey := os.Getenv("DIDIT_API_KEY")

	payload := map[string]interface{}{
		"vendor_data":  uuid,
		"callback_url": "setuhub://callback",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		myErr := utils.ErrorHandler(
			err,
			"Failed to encode request",
			http.StatusInternalServerError,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	reqDidit, err := http.NewRequest(
		http.MethodPost,
		"https://verification.didit.me/v2/session/",
		bytes.NewBuffer(body),
	)
	if err != nil {
		myErr := utils.ErrorHandler(
			err,
			"Failed to create Didit request",
			http.StatusInternalServerError,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	reqDidit.Header.Set("Content-Type", "application/json")
	reqDidit.Header.Set("x-api-key", apiKey)

	client := &http.Client{}

	resp, err := client.Do(reqDidit)
	if err != nil {
		myErr := utils.ErrorHandler(
			err,
			"Failed to connect to Didit",
			http.StatusBadGateway,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	defer resp.Body.Close()

	var result models.DiditSessionResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		myErr := utils.ErrorHandler(
			err,
			"Failed to decode Didit response",
			http.StatusBadGateway,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		myErr := utils.ErrorHandler(
			fmt.Errorf("Didit returned status %d", resp.StatusCode),
			"Didit verification request failed",
			resp.StatusCode,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		myErr := utils.ErrorHandler(
			err,
			"Failed to encode response",
			http.StatusInternalServerError,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}
}

func DiditWebhook(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		myErr := utils.ErrorHandler(
			fmt.Errorf("method not allowed"),
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		myErr := utils.ErrorHandler(
			err,
			"Failed to read webhook body",
			http.StatusBadRequest,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	signature := r.Header.Get("X-Signature-V2")

	if signature == "" {
		myErr := utils.ErrorHandler(
			fmt.Errorf("missing webhook signature"),
			"Missing webhook signature",
			http.StatusUnauthorized,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	// Verify signature here BEFORE decoding the JSON.
	if !verifyDiditSignature(body, signature) {
		myErr := utils.ErrorHandler(
			fmt.Errorf("invalid webhook signature"),
			"Invalid webhook signature",
			http.StatusUnauthorized,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	// NOW it is safe to decode/process the webhook.

	var webhook models.DiditWebhook

	if err := json.Unmarshal(body, &webhook); err != nil {
		myErr := utils.ErrorHandler(
			err,
			"Failed to decode webhook",
			http.StatusBadRequest,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	if webhook.Status != "Approved" {
		w.WriteHeader(http.StatusOK)
		return
	}

	userID := webhook.VendorData

	if userID == "" {
		myErr := utils.ErrorHandler(
			fmt.Errorf("vendor_data is empty"),
			"Missing user ID",
			http.StatusBadRequest,
		)

		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	myErr := databasehandler.AuthenticationDBhandler(
		userID,
		webhook.SessionID,
	)

	if myErr.MyError != nil {
		utils.WriteJSONError(
			w,
			myErr.MyError.Error(),
			myErr.Status,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// check status-----------------------------------------------------------------------
func CheckVerStatus(w http.ResponseWriter, r *http.Request) {

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

	verified, role, myErr := databasehandler.GetVerificationStatusDBHandler(uuid)
	if myErr.MyError != nil {
		utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		return
	}

	if verified {

		tokenString, err := utils.SignToken(uuid, role, "verified")
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
			Status  string `json:"status"`
			Message string `json:"message"`
		}{
			Status:  "Success",
			Message: "Authenticated successfully",
		}

		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}

	} else {
		w.Header().Set("Content-Type", "application/json")

		response := struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		}{
			Status:  "Failed",
			Message: "authentication failed",
		}

		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			myErr := utils.ErrorHandler(err, "Failed to encode response", http.StatusInternalServerError)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}

	}

}
