package databasehandler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
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

	_, err = db.Exec("UPDATE users SET aadhar = $1, name = $2, phone = $3, gender = $4, address = $5, date_of_birth = $6, authentication = $7 WHERE uuid = $8",
		userInfo.Aadhar, userInfo.Name, userInfo.Phone, userInfo.Gender, userInfo.Address, userInfo.DateOfBirth, "verified", uuid,
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

func ProfileInfoDB(uuid string) (models.User, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	var user models.User

	err = db.QueryRow("SELECT uuid, name, phone, gender, address, date_of_birth, authentication, profilePhotoUrl FROM users WHERE uuid = $1", uuid).
		Scan(&user.Uuid, &user.Name, &user.Phone, &user.Gender, &user.Address, &user.DateOfBirth, &user.Authentication, &user.ProfilePhotoURL)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "error retrieving data from database")
	}

	return user, nil
}

// profilePhoto-------------------------------------------------------------------------------------------------------------------------------------------------------------------------
func UpdateProfilePhotoDB(uuid, photoUrl string) error {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	_, err = db.Exec(`UPDATE users SET profile_photo_url = $1 WHERE uuid = $2;`,
		photoUrl, uuid,
	)

	if err != nil {
		return utils.ErrorHandler(err, "error updating profile photo")
	}

	return nil
}

// app handlers-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
func CreateRequestPostDB(uuid, section string, post models.Post, noti chan string) (models.Post, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.Post{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	if post.Radius == 0 {
		post.Radius = 3000
	}

	
	query := getPostAppQuery(section)

	args, err := getPostAppArgs(uuid, section, post)

	if err != nil {
		return models.Post{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if section == "impactevents" {

		err = db.QueryRowContext(ctx, query,
			args...,
		).Scan(&post.PostUUID, &post.CreatedAt, &post.EventAt)

		if err != nil {
			return models.Post{}, utils.ErrorHandler(err, "error inserting post")
		}

	} else {
		err = db.QueryRowContext(ctx, query,
			args...,
		).Scan(&post.PostUUID, &post.CreatedAt)

		if err != nil {
			return models.Post{}, utils.ErrorHandler(err, "error inserting post")
		}

	}

	if section != "moments" {
		go sendNotifications(post, noti)
	}

	return post, nil

}

func RetrieveRequestGetDB(uuid, section string, coordinates models.Coordinates) ([]models.Post, error) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	posts := make([]models.Post, 0)

	query := getGetAppQuery(section)

	args := getGetAppArgs(uuid, section, coordinates)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, query,
		args...,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, utils.ErrorHandler(err, "error making query")
	}
	defer rows.Close()

	for rows.Next() {
		var post models.Post
		var locJSON []byte

		scans := getGetAppScan(section, &post, &locJSON)

		err := rows.Scan(scans...)

		if err != nil {
			return nil, utils.ErrorHandler(err, "error scanning database")
		}

		err = json.Unmarshal(locJSON, &post.Location)
		if err != nil {
			return nil, utils.ErrorHandler(err, fmt.Sprint("error unmarshaling location:"+string(locJSON)))
		}

		posts = append(posts, post)

	}

	return posts, nil
}

func RetrieveMyRequestGetDB(uuid, section string) ([]models.Post, error) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	posts := make([]models.Post, 0)

	query := getGetMyQuery(section)

	args := getGetMyArgs(uuid, section)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, query,
		args...,
	)

	if err != nil {
		return nil, utils.ErrorHandler(err, "error making query")
	}
	defer rows.Close()

	for rows.Next() {
		var post models.Post
		var locJSON []byte

		scans := getGetMyScan(section, &post, &locJSON)

		err := rows.Scan(scans...)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, utils.ErrorHandler(err, "error scanning database")
		}

		err = json.Unmarshal(locJSON, &post.Location)
		if err != nil {
			return nil, utils.ErrorHandler(err, fmt.Sprint("error unmarshaling location:"+string(locJSON)))
		}

		posts = append(posts, post)

	}

	return posts, nil
}

// done --------------------------------------------------------------------------------------------------------------------
func DonePatchRequestDB(postid string) error {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	query := `UPDATE posts SET done = $2 WHERE post_uuid = $1;`

	res, err := db.Exec(query, postid, true)
	if err != nil {
		return utils.ErrorHandler(err, "error updating query")
	}

	// Check if any row was actually updated
	rows, err := res.RowsAffected()
	if err != nil {
		return utils.ErrorHandler(err, "error getting affected rows")
	}

	if rows == 0 {
		return utils.ErrorHandler(fmt.Errorf("no posts with postid %s", postid), "no post found with postid")
	}

	return nil
}

// interested-----------------------------------------------------------------------------------------------------------------------------
func InterestedPostHandler(uuid, postUuid string) (models.InterestResult, error) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.InterestResult{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	var result models.InterestResult

	query := `
	WITH ins AS (
  	  INSERT INTO interested (post_uuid, uuid)
  	  VALUES ($1::uuid, $2::text)
  	  ON CONFLICT DO NOTHING
  	  RETURNING 1
	),
	  upd AS (
  	  UPDATE posts
  	  SET interested_count = interested_count + 1
  	  WHERE post_uuid = $1::uuid
    	AND EXISTS (SELECT 1 FROM ins)
  	  RETURNING interested_count, type
	)
	SELECT
  		EXISTS(SELECT 1 FROM ins) AS changed,
  		COALESCE(u.interested_count, p.interested_count) AS interested_count,
  		p.type
	  FROM posts p
	  LEFT JOIN upd u ON TRUE
	  WHERE p.post_uuid = $1::uuid;
	`

	err = db.QueryRow(query, postUuid, uuid).Scan(&result.Changed, &result.InterestedCount, &result.Type)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.InterestResult{}, utils.ErrorHandler(err, "post not found") // surface as 404 in handler
		}
		return models.InterestResult{}, utils.ErrorHandler(err, "error updating query")
	}

	if result.Changed && result.Type == "helpnearby" {
		go sendNotification(uuid)
	}

	return result, nil
}

func UninterestedPost(uuidStr, postUuid string) (models.InterestResult, error) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.InterestResult{}, utils.ErrorHandler(err, "error connecting to database")
	}
	defer db.Close()

	var result models.InterestResult

	const q = `
	WITH del AS (
	  DELETE FROM interested
	  WHERE post_uuid = $1::uuid AND uuid = $2::text
	  RETURNING 1
	),
	upd AS (
	  UPDATE posts
	  SET interested_count = GREATEST(interested_count - 1, 0)
	  WHERE post_uuid = $1::uuid
	    AND EXISTS (SELECT 1 FROM del)
	  RETURNING interested_count, type
	)
	SELECT
	  EXISTS(SELECT 1 FROM del) AS changed,
	  COALESCE(u.interested_count, p.interested_count) AS interested_count,
	  p.type
	FROM posts p
	LEFT JOIN upd u ON TRUE
	WHERE p.post_uuid = $1::uuid;
	`

	// If the post doesn't exist, this returns sql.ErrNoRows (good -> 404 upstream)
	err = db.QueryRow(q, postUuid, uuidStr).Scan(&result.Changed, &result.InterestedCount, &result.Type)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.InterestResult{}, utils.ErrorHandler(err, "post not found") // surface as 404 in handler
		}
		return models.InterestResult{}, utils.ErrorHandler(err, "error updating query")
	}

	return result, nil
}
