package databasehandler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"hackathon/internal/models"
	nosql "hackathon/internal/repositories/no_sql"
	"hackathon/internal/repositories/sqlconnect"
	"hackathon/pkg/utils"
	"time"
)

var ctx = context.Background()

// signup ------------------------------------------------------------------------------------------------------
func SignUpDBHandler(newUser models.User) (models.User, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	rdb, err := nosql.RedisCliant()

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to (rdb)")
	}

	defer rdb.Close()

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

	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE uuid = $1", newUser.Uuid).Scan(&count)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}

	if count != 0 {
		return models.User{}, utils.ErrorHandler(fmt.Errorf("username already exists in database"), "username already exists")
	}

	otp := randOTP(6)

	if newUser.Role == "" {
		newUser.Role = "user"
	}

	// 	stmt, err := db.Prepare("INSERT INTO users(first_name, last_name, email, class, subject) VALUES(?, ?, ?, ?, ?)")
	key := "data:" + newUser.Uuid

	err = rdb.HSet(ctx, key,
		"otp", otp,
		"email", newUser.Email,
		"pass", newUser.Password,
	).Err()

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error uploading data (rdb)")
	}

	err = rdb.Expire(ctx, key, 7*time.Minute).Err()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error uploading data (rdb)")
	}

	err = sendOTP(newUser.Email, otp)
	if err != nil {
		return models.User{}, err
	}

	newUser.Password = ""
	newUser.ConfirmPassword = ""

	newUser.Authentication = "unverified"

	return newUser, nil

}

// otp--------------------------------------------------------------------------------------------------------------------------

func SignupOtpDBHandler(uuid, role, otp string) (models.User, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	rdb, err := nosql.RedisCliant()

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to (rdb)")
	}

	defer rdb.Close()

	var user models.User

	key := "data:" + uuid

	vals, _ := rdb.HGetAll(ctx, key).Result()

	user.Email = vals["email"]
	user.Password = vals["pass"]
	realOtp := vals["otp"]

	// checking otp ---------------------------------------------------------------------------------
	if otp != realOtp {
		return models.User{}, utils.ErrorHandler(fmt.Errorf("incorrect otp"), "incorrect otp")
	}

	// encoding password-----------------------------------------------------------------------------
	salt := make([]byte, 16)

	_, err = rand.Read(salt)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error making salt")
	}

	user.Password, err = utils.PassEncoder(user.Password, salt)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error encoding pass")
	}

	result, err := db.Exec("INSERT INTO users(uuid, email, password, role, authentication) VALUES($1, $2, $3, $4, $5)",
		uuid, user.Email, user.Password, role, "mail",
	)

	user.Password = ""

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error preparing statement")
	}

	rowsAffected, _ := result.RowsAffected()

	if int(rowsAffected) == 0 {
		return models.User{}, utils.ErrorHandler(err, "no rows effected")
	}

	err = db.QueryRow("SELECT uuid, role, authentication FROM users WHERE uuid = $1", uuid).Scan(
		&user.Uuid, &user.Role, &user.Authentication,
	)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "unable to insert values")
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

	_, err = db.Exec("UPDATE users SET aadhar = $1, name = $2, phone = $3, gender = $4, address = $5, age = $6, authentication = $7 WHERE uuid = $8",
		userInfo.Aadhar, userInfo.Name, userInfo.Phone, userInfo.Gender, userInfo.Address, userInfo.Age, "verified", uuid,
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error updating database")
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
		if err == sql.ErrNoRows {
			return models.User{}, utils.ErrorHandler(err, "email doesnt exists in database")
		}
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
func ForgotPasswordDBHandler(email string) (models.User, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	rdb, err := nosql.RedisCliant()

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to (rdb)")
	}

	defer rdb.Close()

	var user models.User

	err = db.QueryRow("SELECT uuid, role FROM users WHERE email = $1", email).Scan(
		&user.Uuid, &user.Role,
	)

	user.Authentication = "reset"

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error retrieving data from database")
	}

	otp := randOTP(6)

	key := "otp:" + user.Uuid

	err = rdb.Set(ctx, key, otp, 7*time.Minute).Err()

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error uploading data (rdb)")
	}

	err = sendOTP(email, otp)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func ResetPassExecDBHandler(uuid, otp, password string) error {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	rdb, err := nosql.RedisCliant()

	if err != nil {
		return utils.ErrorHandler(err, "error connecting to (rdb)")
	}

	defer rdb.Close()

	key := "otp:" + uuid

	realOtp, err := rdb.Get(ctx, key).Result()

	if err != nil {
		return utils.ErrorHandler(err, "invalid or expired reset code")
	}

	if realOtp != otp {
		return utils.ErrorHandler(err, "otp doesnt match")
	}

	// encoding new password
	salt := make([]byte, 16)

	_, err = rand.Read(salt)
	if err != nil {
		return utils.ErrorHandler(err, "error making salt")
	}
	new_pass, err := utils.PassEncoder(password, salt)
	if err != nil {
		return err
	}

	_, err = db.Exec(`UPDATE users SET password = $1 WHERE uuid = $2`,
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

//-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

func HelpRequestPostDB(uuid string, post models.Post) error {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	result, err := db.Exec("INSERT INTO posts(uuid, type, title, description, coordinates) VALUES($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography);",
		uuid, "help", post.Title, post.Description, post.Longitude, post.Latitude,
	)

	if err != nil {
		return utils.ErrorHandler(err, "error preparing statement")
	}

	rowsAffected, _ := result.RowsAffected()

	if int(rowsAffected) == 0 {
		return utils.ErrorHandler(err, "no rows effected")
	}

	return nil

}

func HelpRequestGetDB(coordinates models.Coordinates) ([]models.Post, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	posts := make([]models.Post, 0)

	cutoff := time.Now().UTC().Add(-30 * time.Minute)
	radius := 500

	rows, err := db.Query("SELECT  post_uuid, title, description, ST_X(coordinates::geometry) AS longitude, ST_Y(coordinates::geometry) AS latitude FROM posts WHERE type = $1 AND ST_DWithin(coordinates, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, $4) AND created_at >= $5;",
		"help", coordinates.Longitude, coordinates.Latitude, radius, cutoff,
	)
	if err != nil {
		return nil, utils.ErrorHandler(err, "error making query")
	}

	for rows.Next() {
		var post models.Post
		err := rows.Scan(&post.PostUUID, &post.Title, &post.Description, &post.Longitude, &post.Latitude)

		if err != nil {
			return nil, utils.ErrorHandler(err, "error scanning database")
		}

		posts = append(posts, post)

	}

	return posts, nil
}
