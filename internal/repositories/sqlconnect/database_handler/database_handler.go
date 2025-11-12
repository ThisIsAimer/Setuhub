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
	"net/http"
	"time"
)

var ctx = context.Background()

// signup ------------------------------------------------------------------------------------------------------
func SignUpDBHandler(newUser models.User) (models.User, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	rdb, err := nosql.RedisCliant()

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error connecting to (rdb)", http.StatusInternalServerError)
	}

	defer rdb.Close()

	// if email exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", newUser.Email).Scan(&count)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error running count query", http.StatusInternalServerError)
	}

	if count != 0 {

		err = db.QueryRow("SELECT uuid, email, password, role, authentication FROM users WHERE email = $1", newUser.Email).Scan(
			&newUser.Uuid, &newUser.Email, &newUser.Password, &newUser.Role, &newUser.Authentication,
		)

		if err != nil {
			return models.User{}, utils.ErrorHandler(err, "User not found", http.StatusInternalServerError)
		}

		err = utils.VerifyPassword(newUser.ConfirmPassword, newUser.Password)

		if err != nil {
			return models.User{}, utils.ErrorHandler(err, "Account already exists, your password is invalid", http.StatusBadRequest)
		}

		newUser.Password = ""
		newUser.ConfirmPassword = ""

		return newUser, utils.Errorhandler{}

	}

	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE uuid = $1", newUser.Uuid).Scan(&count)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}

	if count != 0 {
		return models.User{}, utils.ErrorHandler(fmt.Errorf("username already exists in database"), "Username already exists", http.StatusBadRequest)
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
		return models.User{}, utils.ErrorHandler(err, "Error uploading data (rdb)", http.StatusInternalServerError)
	}

	err = rdb.Expire(ctx, key, 7*time.Minute).Err()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error uploading data (rdb)", http.StatusInternalServerError)
	}

	err = sendOTP(newUser.Email, otp)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error sending otp", http.StatusInternalServerError)
	}

	newUser.Password = ""
	newUser.ConfirmPassword = ""

	newUser.Authentication = "unverified"

	return newUser, utils.Errorhandler{}

}

// otp--------------------------------------------------------------------------------------------------------------------------

func SignupOtpDBHandler(uuid, role, otp string) (models.User, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	rdb, err := nosql.RedisCliant()

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error connecting to (rdb)", http.StatusInternalServerError)
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
		return models.User{}, utils.ErrorHandler(fmt.Errorf("incorrect otp"), "Incorrect otp", http.StatusBadRequest)
	}

	// encoding password-----------------------------------------------------------------------------
	salt := make([]byte, 16)

	_, err = rand.Read(salt)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error making salt", http.StatusInternalServerError)
	}

	user.Password, err = utils.PassEncoder(user.Password, salt)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error encoding pass", http.StatusInternalServerError)
	}

	result, err := db.Exec("INSERT INTO users(uuid, email, password, role, authentication) VALUES($1, $2, $3, $4, $5)",
		uuid, user.Email, user.Password, role, "mail",
	)

	user.Password = ""

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error preparing statement", http.StatusInternalServerError)
	}

	rowsAffected, _ := result.RowsAffected()

	if int(rowsAffected) == 0 {
		return models.User{}, utils.ErrorHandler(err, "No rows effected", http.StatusInternalServerError)
	}

	err = db.QueryRow("SELECT uuid, role, authentication FROM users WHERE uuid = $1", uuid).Scan(
		&user.Uuid, &user.Role, &user.Authentication,
	)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Unable to retrieve data", http.StatusInternalServerError)
	}

	return user, utils.Errorhandler{}
}

// authenticat ------------------------------------------------------------------------------------------------------------------

func AuthenticationDBhandler(uuid string, userInfo models.UserInfo) (models.User, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	_, err = db.Exec("UPDATE users SET aadhar = $1, name = $2, phone = $3, gender = $4, address = $5, date_of_birth = $6, authentication = $7 WHERE uuid = $8",
		userInfo.Aadhar, userInfo.Name, userInfo.Phone, userInfo.Gender, userInfo.Address, userInfo.DateOfBirth, "verified", uuid,
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error updating database", http.StatusInternalServerError)
	}

	var user models.User

	err = db.QueryRow("SELECT uuid, role, authentication FROM users WHERE uuid = $1", uuid).Scan(
		&user.Uuid, &user.Role, &user.Authentication,
	)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "User not found", http.StatusInternalServerError)
	}

	return user, utils.Errorhandler{}
}

// login---------------------------------------------------------------------------------------------------------------------------
func LoginDBHandlerFunc(email, givenPass string) (models.User, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	var user models.User

	err = db.QueryRow("SELECT uuid, password, role, authentication FROM users WHERE email = $1", email).Scan(
		&user.Uuid, &user.Password, &user.Role, &user.Authentication,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, utils.ErrorHandler(err, "Invalid email or password", http.StatusBadRequest)
		}
		return models.User{}, utils.ErrorHandler(err, "Error retrieving data from database", http.StatusInternalServerError)
	}

	err = utils.VerifyPassword(givenPass, user.Password)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Invalid email or password", http.StatusBadRequest)
	}

	user.Password = ""

	return user, utils.Errorhandler{}
}

// forgot password------------------------------------------------------------------------------------------------------------
func ForgotPasswordDBHandler(email string) (models.User, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	rdb, err := nosql.RedisCliant()

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error connecting to (rdb)", http.StatusInternalServerError)
	}

	defer rdb.Close()

	var user models.User

	err = db.QueryRow("SELECT uuid, role FROM users WHERE email = $1", email).Scan(
		&user.Uuid, &user.Role,
	)

	user.Authentication = "reset"

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error retrieving data from database", http.StatusInternalServerError)
	}

	otp := randOTP(6)

	key := "otp:" + user.Uuid

	err = rdb.Set(ctx, key, otp, 7*time.Minute).Err()

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error uploading data (rdb)", http.StatusInternalServerError)
	}

	err = sendOTP(email, otp)
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error sending otp", http.StatusInternalServerError)
	}

	return user, utils.Errorhandler{}
}

func ResetPassExecDBHandler(uuid, otp, password string) utils.Errorhandler {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	rdb, err := nosql.RedisCliant()

	if err != nil {
		return utils.ErrorHandler(err, "Error connecting to (rdb)", http.StatusInternalServerError)
	}

	defer rdb.Close()

	key := "otp:" + uuid

	realOtp, err := rdb.Get(ctx, key).Result()

	if err != nil {
		return utils.ErrorHandler(err, "Invalid or expired reset code", http.StatusBadRequest)
	}

	if realOtp != otp {
		return utils.ErrorHandler(err, "Invalid Otp", http.StatusBadRequest)
	}

	// encoding new password
	salt := make([]byte, 16)

	_, err = rand.Read(salt)
	if err != nil {
		return utils.ErrorHandler(err, "Error making salt", http.StatusInternalServerError)
	}
	new_pass, err := utils.PassEncoder(password, salt)
	if err != nil {
		return utils.ErrorHandler(err, "Error encoding password", http.StatusInternalServerError)
	}

	_, err = db.Exec(`UPDATE users SET password = $1 WHERE uuid = $2`,
		new_pass, uuid,
	)

	if err != nil {
		return utils.ErrorHandler(err, "Error updating password", http.StatusInternalServerError)
	}

	return utils.Errorhandler{}
}

// storesLocation ---------------------------------------------------------------------------------------------------

func UpdateCoordinates(uuid string, coordinates models.Coordinates) utils.Errorhandler {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	_, err = db.Exec(`UPDATE users SET coordinates = ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography WHERE uuid = $3;`,
		coordinates.Longitude, coordinates.Latitude, uuid,
	)

	if err != nil {
		return utils.ErrorHandler(err, "Error updating coordinates", http.StatusInternalServerError)
	}

	return utils.Errorhandler{}
}

func ProfileInfoDB(uuid string) (models.User, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	var user models.User

	err = db.QueryRow("SELECT uuid, name, phone, gender, address, date_of_birth, authentication, profilePhotoUrl FROM users WHERE uuid = $1", uuid).
		Scan(&user.Uuid, &user.Name, &user.Phone, &user.Gender, &user.Address, &user.DateOfBirth, &user.Authentication, &user.ProfilePhotoURL)

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error retrieving data from database", http.StatusInternalServerError)
	}

	return user, utils.Errorhandler{}
}

// profilePhoto-------------------------------------------------------------------------------------------------------------------------------------------------------------------------
func UpdateProfilePhotoDB(uuid, photoUrl string) utils.Errorhandler {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	_, err = db.Exec(`UPDATE users SET profile_photo_url = $1 WHERE uuid = $2;`,
		photoUrl, uuid,
	)

	if err != nil {
		return utils.ErrorHandler(err, "Error updating profile photo", http.StatusInternalServerError)
	}

	return utils.Errorhandler{}
}

// app handlers-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
func CreateRequestPostDB(uuid, section string, post models.Post, noti chan string) (models.Post, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.Post{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	if post.Radius == 0 {
		post.Radius = 3000
	}

	query := getPostAppQuery(section)

	args, err := getPostAppArgs(uuid, section, post)

	if err != nil {
		return models.Post{}, utils.ErrorHandler(err, err.Error(), http.StatusBadRequest)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if section == "impactevents" {

		err = db.QueryRowContext(ctx, query,
			args...,
		).Scan(&post.PostUUID, &post.CreatedAt, &post.EventAt)

		if err != nil {
			return models.Post{}, utils.ErrorHandler(err, "Error inserting post", http.StatusInternalServerError)
		}

	} else {
		err = db.QueryRowContext(ctx, query,
			args...,
		).Scan(&post.PostUUID, &post.CreatedAt)

		if err != nil {
			return models.Post{}, utils.ErrorHandler(err, "Error inserting post", http.StatusInternalServerError)
		}

	}

	if section != "moments" {
		go sendNotifications(post, noti)
	}

	return post, utils.Errorhandler{}

}

// CreateButtonHandler ----------------------------------------------------------------------------------------------------------------------------------------

func CreatePostButtonDbHandle(uuid string) (models.User, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	var user models.User

	err = db.QueryRow(`SELECT name, profile_photo_url FROM users WHERE uuid = $1`, uuid).
		Scan(&user.Name, &user.ProfilePhotoURL)

	user.Uuid = uuid

	if err != nil {
		return models.User{}, utils.ErrorHandler(err, "Error retrieving data from database", http.StatusInternalServerError)
	}

	return user, utils.Errorhandler{}
}

// app handlers -----------------------------------------------------------------------------------------------------------------------------------------------------------
func RetrieveRequestGetDB(uuid, section string, page int, coordinates models.Coordinates) ([]models.Post, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	posts := make([]models.Post, 0)

	query := getGetAppQuery(section)

	args := getGetAppArgs(uuid, section, page, coordinates)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, query,
		args...,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, utils.Errorhandler{}
		}
		return nil, utils.ErrorHandler(err, "Error making query", http.StatusInternalServerError)
	}
	defer rows.Close()

	for rows.Next() {
		var post models.Post
		var locJSON []byte

		scans := getGetAppScan(section, &post, &locJSON)

		err := rows.Scan(scans...)

		if err != nil {
			return nil, utils.ErrorHandler(err, "Error scanning database", http.StatusInternalServerError)
		}

		err = json.Unmarshal(locJSON, &post.Location)
		if err != nil {
			return nil, utils.ErrorHandler(err, fmt.Sprint("Error unmarshaling location:"+string(locJSON)), http.StatusInternalServerError)
		}

		posts = append(posts, post)

		if err := rows.Err(); err != nil {
			return nil, utils.ErrorHandler(err, "Row iteration error", http.StatusInternalServerError)
		}

	}

	return posts, utils.Errorhandler{}
}

func RetrieveMyRequestGetDB(uuid, section string, page int) ([]models.Post, utils.Errorhandler) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	posts := make([]models.Post, 0)

	query := getGetMyQuery(section)

	args := getGetMyArgs(uuid, section, page)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, query,
		args...,
	)

	if err != nil {
		return nil, utils.ErrorHandler(err, "Error making query", http.StatusInternalServerError)
	}
	defer rows.Close()

	for rows.Next() {
		var post models.Post
		var locJSON []byte

		scans := getGetMyScan(section, &post, &locJSON)

		err := rows.Scan(scans...)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, utils.Errorhandler{}
			}
			return nil, utils.ErrorHandler(err, "Error scanning database", http.StatusInternalServerError)
		}

		err = json.Unmarshal(locJSON, &post.Location)
		if err != nil {
			return nil, utils.ErrorHandler(err, fmt.Sprint("Error unmarshaling location:"+string(locJSON)), http.StatusInternalServerError)
		}

		posts = append(posts, post)

		if err := rows.Err(); err != nil {
			return nil, utils.ErrorHandler(err, "Row iteration error", http.StatusInternalServerError)
		}

	}

	return posts, utils.Errorhandler{}
}

// expo token--------------------------------------------------------------------

func SetFirebaseTokenDbHandler(uuid, token string) utils.Errorhandler {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	query := `UPDATE users SET firebase_token = $1 WHERE uuid = $2;`

	res, err := db.Exec(query, token, uuid)
	if err != nil {
		return utils.ErrorHandler(err, "Error updating query", http.StatusInternalServerError)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return utils.ErrorHandler(err, "Error getting affected rows", http.StatusInternalServerError)
	}

	if rows == 0 {
		return utils.ErrorHandler(fmt.Errorf("invalid uuid"), "Invalid uuid", http.StatusBadRequest)
	}

	return utils.Errorhandler{}
}

// done --------------------------------------------------------------------------------------------------------------------
func DonePatchRequestDB(postid string) utils.Errorhandler {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	query := `UPDATE posts SET done = $2 WHERE post_uuid = $1;`

	res, err := db.Exec(query, postid, true)
	if err != nil {
		return utils.ErrorHandler(err, "Error updating query", http.StatusInternalServerError)
	}

	// Check if any row was actually updated
	rows, err := res.RowsAffected()
	if err != nil {
		return utils.ErrorHandler(err, "Error getting affected rows", http.StatusInternalServerError)
	}

	if rows == 0 {
		return utils.ErrorHandler(fmt.Errorf("no posts with postid %s", postid), "No post found with postid", http.StatusBadRequest)
	}

	return utils.Errorhandler{}
}

// interested-----------------------------------------------------------------------------------------------------------------------------
func InterestedPostHandler(uuid, postUuid string) (models.InterestResult, utils.Errorhandler) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.InterestResult{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
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
			return models.InterestResult{}, utils.ErrorHandler(err, "Post not found", http.StatusInternalServerError) // surface as 404 in handler
		}
		return models.InterestResult{}, utils.ErrorHandler(err, "Error updating query", http.StatusInternalServerError)
	}

	if result.Changed && result.Type == "helpnearby" {
		go sendNotification(uuid)
	}

	return result, utils.Errorhandler{}
}

func UninterestedPost(uuidStr, postUuid string) (models.InterestResult, utils.Errorhandler) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.InterestResult{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
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
			return models.InterestResult{}, utils.ErrorHandler(err, "Post not found", http.StatusInternalServerError) // surface as 404 in handler
		}
		return models.InterestResult{}, utils.ErrorHandler(err, "Error updating query", http.StatusInternalServerError)
	}

	return result, utils.Errorhandler{}
}

// comments------------------------------------------------------------------------------------------------------------------

func GetCommentDBHandler(postUuid, uuid string, page int) ([]models.Comment, utils.Errorhandler) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	limit := 20

	offset := (page - 1) * limit

	query := `
 	SELECT c.comment_uuid, c.uuid, c.content, c.edited, c.created_at, u.profile_photo_url
	  FROM comments AS c
	  JOIN users AS u ON c.uuid = u.uuid
	  WHERE c.post_uuid = $1::uuid
	  ORDER BY (uuid = $2::text) DESC, created_at DESC, comment_uuid DESC
	LIMIT $3::int OFFSET $4;
  `
	rows, err := db.Query(query, postUuid, uuid, limit, offset)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, utils.Errorhandler{}
		}
		return nil, utils.ErrorHandler(err, "Error making query", http.StatusInternalServerError)
	}
	defer rows.Close()

	comments := make([]models.Comment, 0)

	for rows.Next() {
		var comment models.Comment

		err := rows.Scan(&comment.CommentUUID, &comment.Uuid, &comment.Content, &comment.Edited, &comment.CreatedAt, &comment.ProfilePhotoURL)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, utils.Errorhandler{}
			}
			return nil, utils.ErrorHandler(err, "Error scanning database", http.StatusInternalServerError)
		}

		comments = append(comments, comment)

		if err := rows.Err(); err != nil {
			return nil, utils.ErrorHandler(err, "Row iteration error", http.StatusInternalServerError)
		}

	}

	return comments, utils.Errorhandler{}
}

func CreateCommentDBHandler(comment models.Comment) (models.Comment, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return models.Comment{}, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	query := `
	WITH ins AS (
	  INSERT INTO comments (post_uuid, uuid, content)
  	  VALUES ($1::uuid, $2::text, $3::text)
  	  RETURNING comment_uuid, created_at
	),
	upd AS (
  	  UPDATE posts
  	  SET comment_count = comment_count + 1
  	    WHERE post_uuid = $1::uuid
	  RETURNING comment_count
	)
	SELECT
  	  i.comment_uuid, i.created_at, u.comment_count
	FROM ins i
	JOIN upd u ON TRUE;
	`
	err = db.QueryRow(query, comment.PostUUID, comment.Uuid, comment.Content).
		Scan(&comment.CommentUUID, &comment.CreatedAt, &comment.CommentCount)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.Comment{}, utils.ErrorHandler(err, "Post not found", http.StatusInternalServerError) // surface as 404 in handler
		}
		return models.Comment{}, utils.ErrorHandler(err, "Error updating query", http.StatusInternalServerError)
	}

	return comment, utils.Errorhandler{}
}

func EditCommentDBHandler(commentUuid, uuid, content string) (bool, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return false, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	var edited bool

	query := `
	UPDATE comments
	  SET content = $3::text,
      edited = true
	WHERE comment_uuid = $1::uuid
  	  AND uuid = $2::text
	RETURNING edited;
	`

	err = db.QueryRow(query, commentUuid, uuid, content).
		Scan(&edited)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, utils.ErrorHandler(err, "Comment not found or not owned", http.StatusBadRequest)
		}
		return false, utils.ErrorHandler(err, "Error updating query", http.StatusInternalServerError)
	}

	return edited, utils.Errorhandler{}
}

func DeleteCommentDBHandler(commentUuid, uuid string) (bool, int, utils.Errorhandler) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		return false, 0, utils.ErrorHandler(err, "Error connecting to database", http.StatusInternalServerError)
	}
	defer db.Close()

	var deleted bool
	var commentCount int

	query := `
	WITH del AS (
	  DELETE FROM comments
	  WHERE comment_uuid = $1::uuid
		AND uuid = $2::text
	  RETURNING post_uuid
	),
	upd AS (
	  UPDATE posts p
	  SET comment_count = GREATEST(p.comment_count - 1, 0)
	    WHERE p.post_uuid = (SELECT post_uuid FROM del)
	  RETURNING p.post_uuid, p.comment_count, p.type
	)
	SELECT EXISTS(SELECT 1 FROM del) AS deleted,
	COALESCE((SELECT comment_count FROM upd), 0)
	FROM upd u;
	`
	err = db.QueryRow(query, commentUuid, uuid).
		Scan(&deleted, &commentCount)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, 0, utils.ErrorHandler(err, "Comment not found", http.StatusInternalServerError) // surface as 404 in handler
		}
		return false, 0, utils.ErrorHandler(err, "Error updating query", http.StatusInternalServerError)
	}

	return deleted, commentCount, utils.Errorhandler{}
}
