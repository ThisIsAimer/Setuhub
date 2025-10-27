package databasehandler

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"hackathon/internal/models"
	"hackathon/internal/repositories/sqlconnect"
	"hackathon/pkg/utils"

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

		err = db.QueryRow("SELECT uuid, email, password, role, authentication, otp FROM users WHERE email = $1", newUser.Email).Scan(
			&newUser.Uuid, &newUser.Email, &newUser.Password, &newUser.Role, &newUser.Authentication, &newUser.Otp,
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

		if newUser.Authentication == "mail" || newUser.Authentication == "verified" {
			return newUser, nil
		}

		myMail := mail.NewMessage()

		myMail.SetHeader("From", "ourapp@example.com") // replace email
		myMail.SetHeader("To", newUser.Email)
		myMail.SetHeader("Subject", "OTP For our app")
		myMail.SetBody("text/plain", "your OTP for our app is: "+newUser.Otp.String)

		dialer := mail.NewDialer("localhost", 1025, "", "")
		err = dialer.DialAndSend(myMail)
		if err != nil {
			return models.User{}, utils.ErrorHandler(err, "error sending mail")
		}

		newUser.Otp.String = ""

		return newUser, nil

	}

	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE uuid = $1", newUser.Uuid).Scan(&count)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}

	if count != 0 {
		return models.User{}, utils.ErrorHandler(fmt.Errorf("username already exists in database"), "username already exists")
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
	result, err := db.Exec("INSERT INTO users(uuid, email, password, role, otp, authentication) VALUES($1, $2, $3, $4, $5 $6)",
		newUser.Uuid, newUser.Email, newUser.Password, newUser.Role, otp, "unverified",
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

	newUser.Password = ""
	newUser.ConfirmPassword = ""

	if newUser.Authentication == "mail" || newUser.Authentication == "verified" {
		return newUser, nil
	}

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

	var user models.User

	err = db.QueryRow("SELECT otp FROM users WHERE uuid = $1", uuid).Scan(
		&user.Otp,
	)

	if !user.Otp.Valid {
		return models.User{}, utils.ErrorHandler(fmt.Errorf("no Otp present in database"), "no Otp present in database")
	}

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "user not found")
	}

	if otp != user.Otp.String {
		return models.User{}, utils.ErrorHandler(fmt.Errorf("incorrect otp"), "incorrect otp")
	}

	_, err = db.Exec("UPDATE users SET authentication = $1, otp = NULL WHERE uuid = $2",
		"mail", uuid,
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error setting token")
	}

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

	_, err = db.Exec("UPDATE users SET aadhar = $1, name = $2 phone = $3, gender = $4, address = $5, age = $6, authentication = $7 WHERE uuid = $8",
		userInfo.Aadhar, userInfo.Name, userInfo.Phone, userInfo.Gender, userInfo.Address, userInfo.Age, "verified", uuid,
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

	err = db.QueryRow("SELECT uuid, password, role, authentication, otp FROM users WHERE email = $1", email).Scan(
		&user.Uuid, &user.Password, &user.Role, &user.Authentication, &user.Otp,
	)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error retrieving data from database")
	}

	err = utils.VerifyPassword(givenPass, user.Password)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "passwords dont match")
	}

	user.Password = ""

	if user.Authentication == "mail" || user.Authentication == "verified" {
		return user, nil
	}

	myMail := mail.NewMessage()

	myMail.SetHeader("From", "ourapp@example.com") // replace email
	myMail.SetHeader("To", email)
	myMail.SetHeader("Subject", "OTP For our app")
	myMail.SetBody("text/plain", "your OTP for our app is: "+user.Otp.String)

	dialer := mail.NewDialer("localhost", 1025, "", "")
	err = dialer.DialAndSend(myMail)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error sending mail")
	}

	user.Otp.String = ""

	return user, nil
}

// forgot password------------------------------------------------------------------------------------------------------------
func ForgotPasswordDBHandler(email string) (models.User, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	var user models.User

	err = db.QueryRow("SELECT uuid, role FROM users WHERE email = $1", email).Scan(
		&user.Uuid, &user.Role,
	)

	user.Authentication = "reset"

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error retrieving data from database")
	}

	otp := randOTP(6)

	_, err = db.Exec("UPDATE users SET otp = $1 WHERE uuid = $2",
		otp, user.Uuid,
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error setting token")
	}

	message := fmt.Sprintf("forgot your password for <our app>? OTP to reset password is: %s", otp)

	myMail := mail.NewMessage()

	myMail.SetHeader("From", "ourapp@example.com") // replace email
	myMail.SetHeader("To", email)
	myMail.SetHeader("Subject", "Password reset otp")
	myMail.SetBody("text/plain", message)

	dialer := mail.NewDialer("localhost", 1025, "", "")
	err = dialer.DialAndSend(myMail)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error sending mail")
	}

	return user, nil
}

func ResetPassExecDBHandler(uuid, otp, password string) error {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	var realOtp sql.NullString

	err = db.QueryRow(`Select otp FROM users WHERE uuid = $1`, uuid).
		Scan(&realOtp)

	if !realOtp.Valid {
		return utils.ErrorHandler(fmt.Errorf("no Otp present in database"), "no Otp present in database")
	}

	if err != nil {
		return utils.ErrorHandler(err, "invalid or expired reset code")
	}

	salt := make([]byte, 16)

	_, err = rand.Read(salt)
	if err != nil {
		return utils.ErrorHandler(err, "error making salt")
	}
	new_pass, err := utils.PassEncoder(password, salt)
	if err != nil {
		return err
	}

	_, err = db.Exec(`UPDATE users SET password = $1, otp = NULL WHERE uuid = $2`,
		new_pass, uuid,
	)

	if err != nil {
		return utils.ErrorHandler(err, "error updating password")
	}

	return nil
}

// storesLocation ---------------------------------------------------------------------------------------------------

func UpdateCoordinates(uuid string, coordinates models.Coordinates) error {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	_, err = db.Exec(`UPDATE users SET coordinates = ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography WHERE uuid = $3;`,
		coordinates.Longitude, coordinates.Latitude, uuid,
	)

	if err != nil {
		return utils.ErrorHandler(err, "error updating coordinates")
	}

	return nil
}
