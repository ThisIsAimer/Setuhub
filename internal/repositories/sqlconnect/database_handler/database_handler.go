package databasehandler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hackathon/internal/models"
	"hackathon/internal/repositories/sqlconnect"
	"hackathon/pkg/utils"
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

	// if email exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", newUser.Email).Scan(&count)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error running count query")
	}

	if count != 0 {
		err = db.QueryRow("SELECT uuid, email, password, role, authentication FROM users WHERE email = $1", newUser.Email).Scan(
			&newUser.Uuid, &newUser.Email, &newUser.Password, &newUser.Role, &newUser.Authentication,
		)

		if err != nil {
			return models.User{}, utils.ErrorHandler(err, "user not found")
		}

		err = utils.VerifyPassword(newUser.ConfirmPassword, newUser.Password)

		if err != nil {
			return models.User{}, utils.ErrorHandler(err, "account exists but password wrong")
		}

		newUser.Password = ""
		newUser.ConfirmPassword = ""

		return newUser, nil
	}

	// if user not exist
	salt := make([]byte, 16)

	_, err = rand.Read(salt)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error making salt")
	}

	newUser.Password, err = utils.PassEncoder(newUser.Password, salt)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error encoding pass")
	}

	otp := randOTP(6)

	if newUser.Role == "" {
		newUser.Role = "user"
	}

	// 	stmt, err := db.Prepare("INSERT INTO users(first_name, last_name, email, class, subject) VALUES(?, ?, ?, ?, ?)")
	result, err := db.Exec("INSERT INTO users(email, password, role, otp, authentication) VALUES($1, $2, $3, $4, $5)",
		newUser.Email, newUser.Password, newUser.Role, otp, "unverified",
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error preparing statement")
	}

	rowsAffected, _ := result.RowsAffected()

	if int(rowsAffected) == 0 {
		return models.User{}, utils.ErrorHandler(err, "no rows effected")
	}

	err = db.QueryRow("SELECT uuid, email, password, role, authentication FROM users WHERE email = $1", newUser.Email).Scan(
		&newUser.Uuid, &newUser.Email, &newUser.Password, &newUser.Role, &newUser.Authentication,
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "user not found")
	}

	if newUser.Authentication == "mail" || newUser.Authentication == "verified" {
		return newUser, nil
	}

	newUser.Password = ""
	newUser.ConfirmPassword = ""

	myMail := mail.NewMessage()

	myMail.SetHeader("From", "ourapp@example.com") // replace email
	myMail.SetHeader("To", newUser.Email)
	myMail.SetHeader("Subject", "OTP For our app")
	myMail.SetBody("text/plain", "your OTP for our app is: "+otp)

	dialer := mail.NewDialer("localhost", 1025, "", "")
	err = dialer.DialAndSend(myMail)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error sending mail")
	}

	return newUser, nil

}

// otp--------------------------------------------------------------------------------------------------------------------------

func SignupOtpDBHandler(uuid, otp string) (models.User, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	var realOtp string

	err = db.QueryRow("SELECT otp, authentication FROM users WHERE uuid = $1", otp).Scan(
		&realOtp,
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "user not found")
	}

	if otp != realOtp {
		return models.User{}, utils.ErrorHandler(fmt.Errorf("incorrect otp"), "incorrect otp")
	}

	_, err = db.Exec("UPDATE users SET authentication = $1, otp = $2 WHERE uuid = $3",
		"mail", "", uuid,
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error setting token")
	}

	var user models.User

	err = db.QueryRow("SELECT uuid, role, authentication FROM users WHERE uuid = $1", uuid).Scan(
		&user.Uuid, &user.Role, &user.Authentication,
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "user not found")
	}

	return user, nil
}

// authenticat ------------------------------------------------------------------------------------------------------------------

func AuthenticationDBhandler(uuid string, userInfo models.UserInfo) (models.User, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	_, err = db.Exec("UPDATE users SET aadhar = $1, phone = $2, gender = $3, address = $4, age = $5, authentication = $6 WHERE uuid = $7",
		userInfo.Aadhar, userInfo.Phone, userInfo.Gender, userInfo.Address, userInfo.Age, "verified", uuid,
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error setting token")
	}

	var user models.User

	err = db.QueryRow("SELECT uuid, role, authentication FROM users WHERE uuid = $1", uuid).Scan(
		&user.Uuid, &user.Role, &user.Authentication,
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "user not found")
	}

	return user, nil
}

// login---------------------------------------------------------------------------------------------------------------------------
func LoginDBHandlerFunc(email, givenPass string) (models.User, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	var user models.User

	err = db.QueryRow("SELECT uuid, password, role, authentication FROM users WHERE email = $1", email).Scan(
		&user.Uuid, &user.Password, &user.Role, &user.Authentication,
	)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error retrieving data from database")
	}

	err = utils.VerifyPassword(givenPass, user.Password)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "passwords dont match")
	}

	user.Password = ""

	return user, nil
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

	myMail.SetHeader("From", "ourapp@example.com") // replace email
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
