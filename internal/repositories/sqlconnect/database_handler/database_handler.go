package databasehandler

import (
	"hackathon/internal/models"
	"hackathon/internal/repositories/sqlconnect"
	"hackathon/pkg/utils"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-mail/mail/v2"
)

// signup ------------------------------------------------------------------------------------------------------
func SignUpDBHandler(newUser models.User) (models.User, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	salt := make([]byte, 16)

	_, err = rand.Read(salt)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error making salt")
	}

	newUser.Password, err = utils.PassEncoder(newUser.Password, salt)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error encoding pass")
	}

	// 	stmt, err := db.Prepare("INSERT INTO users(first_name, last_name, email, class, subject) VALUES(?, ?, ?, ?, ?)")
	result, err := db.Exec("INSERT INTO users(email, password, role) VALUES($1, $2, $3)", newUser.Email, newUser.Password, newUser.Role)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error preparing statement")
	}

	rowsAffected, _ := result.RowsAffected()

	db.QueryRow("SELECT uuid, email, password, role, authentication FROM users WHERE email = $1",newUser.Email).Scan(
		&newUser.Uuid, &newUser.Email, &newUser.Password, &newUser.Role, &newUser.Authentication,
	)
	newUser.Password = ""
	newUser.ConfirmPassword = ""

	if int(rowsAffected) == 0 {
		return models.User{}, utils.ErrorHandler(err, "no rows effected")
	}

	return newUser, nil

}

// login---------------------------------------------------------------------------------------------------------------------------
func LoginDBHandlerFunc(givenPass string, email string) (models.User, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	var dbUser models.User

	err = db.QueryRow("SELECT uuid, email, password, role, authentication FROM users WHERE email = $1", email).Scan(
		&dbUser.Uuid, &dbUser.Email, &dbUser.Password,
	)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error retrieving data from database")
	}

	return dbUser, nil
}

// forgot password------------------------------------------------------------------------------------------------------------
func ForgotPasswordDBHandler(email string) error {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	var user models.User

	err = db.QueryRow("SELECT uuid FROM users WHERE email = $1", email).Scan(&user.Uuid)

	if err != nil {
		return utils.ErrorHandler(err, "user not found")
	}

	expResetTime, err := strconv.Atoi(os.Getenv("RESET_TOKEN_EXP_DURATION"))
	if err != nil {
		return utils.ErrorHandler(err, "failed to send password reset mail")
	}

	mins := time.Duration(expResetTime) * time.Minute

	// adding expiry time
	expiry := time.Now().Add(mins)

	tokenBytes := make([]byte, 32)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		return utils.ErrorHandler(err, "error making salt")
	}

	token := hex.EncodeToString(tokenBytes)

	hashedToken := sha256.Sum256(tokenBytes)

	hashedTokenString := hex.EncodeToString(hashedToken[:])
	fmt.Println("expiry time", expiry)

	_, err = db.Exec("UPDATE users SET password_reset_code = $1, password_reset_expires = $2 WHERE uuid = $3", hashedTokenString, expiry, user.Uuid)

	if err != nil {
		return utils.ErrorHandler(err, "error setting token")
	}

	baseURL := os.Getenv("BASE_URL")

	resetUrl := fmt.Sprintf("%s/login/forgotpassword/reset/%s", baseURL, token)
	message := fmt.Sprintf(" forgot your password? reset it using link %s \nIf you didnt reset a password reset, please ignore, the link is only valid for %v mins", resetUrl, expiry)

	myMail := mail.NewMessage()

	myMail.SetHeader("From", "smth@school.com") // replace email
	myMail.SetHeader("To", email)
	myMail.SetHeader("Subject", "Password reset link")
	myMail.SetBody("text/plain", message)

	dialer := mail.NewDialer("localhost", 1025, "", "")
	err = dialer.DialAndSend(myMail)
	if err != nil {
		return utils.ErrorHandler(err, "error sending mail")
	}

	return nil
}

func ResetPassExecDBHandler(resetCode, new_pass string) error {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	var user models.User

	bytes, err := hex.DecodeString(resetCode)
	if err != nil {
		return utils.ErrorHandler(err, "error decoding string")
	}

	hashedToken := sha256.Sum256(bytes)

	hashedTokenString := hex.EncodeToString(hashedToken[:])

	query := `Select uuid, email FROM users WHERE password_reset_code = $1 AND password_reset_expires > $2`

	err = db.QueryRow(query, hashedTokenString, time.Now()).
		Scan(&user.Uuid, &user.Email)

	if err != nil {
		return utils.ErrorHandler(err, "invalid or expired reset code")
	}

	salt := make([]byte, 16)

	_, err = rand.Read(salt)
	if err != nil {
		return utils.ErrorHandler(err, "error making salt")
	}
	new_pass, err = utils.PassEncoder(new_pass, salt)
	if err != nil {
		return err
	}

	updateQuery := `UPDATE users SET password = $1, password_reset_code = NULL, password_reset_expires = NULL WHERE uuid = $2`

	_, err = db.Exec(updateQuery, new_pass, user.Uuid)

	if err != nil {
		return utils.ErrorHandler(err, "error updating password")
	}

	return nil
}
