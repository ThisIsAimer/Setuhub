package databasehandler

import (
	"encoding/json"
	"fmt"
	"hackathon/internal/models"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lib/pq"
)

func randOTP(n int) string {
	var letters = []rune("0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func sendOTP(to, otp string) error {
	apiKey := os.Getenv("BREVO_API_KEY")
	from := os.Getenv("FROM_EMAIL")
	fromName := os.Getenv("FROM_NAME") // optional

	if apiKey == "" || from == "" {
		return fmt.Errorf("missing brevo api credentials")
	}

	payload := fmt.Sprintf(`{
	  "sender": {
		"email": "%s",
		"name": "%s"
	  },
	  "to": [
		{ "email": "%s" }
	  ],
	  "subject": "Your OTP Code",
	  "htmlContent": "<p>Your OTP is <b>%s</b><br/>Expires in 7 minutes.</p>"
	}`, from, fromName, to, otp)

	req, err := http.NewRequest(
		http.MethodPost,
		"https://api.brevo.com/v3/smtp/email",
		strings.NewReader(payload),
	)
	if err != nil {
		return err
	}

	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("brevo api error (%d): %s", resp.StatusCode, body)
	}

	return nil
}

func getPostAppQuery(section string) string {
	query := make(map[string]string, 0)

	query["helpnearby"] = "INSERT INTO posts(uuid, type, title, description, coordinates, radius, location) VALUES($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography, $7, $8::jsonb) RETURNING post_uuid, created_at;"
	query["impactevents"] = "INSERT INTO posts(uuid, type, title, description,  media, coordinates, event_at, radius, location) VALUES($1, $2, $3, $4, $5::text[], ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography, $8, $9, $10::jsonb) RETURNING post_uuid, created_at, event_at;"
	query["moments"] = "INSERT INTO posts(uuid, type, title, description, media) VALUES($1, $2, $3, $4, $5::text[]) RETURNING post_uuid, created_at;"
	query["missingpeople"] = "INSERT INTO posts(uuid, type, title, description, gender, age,  media, coordinates, radius, location) VALUES($1, $2, $3, $4, $5, $6,  $7::text[], ST_SetSRID(ST_MakePoint($8, $9), 4326)::geography, $10, $11::jsonb) RETURNING post_uuid, created_at;"
	query["bloodemergency"] = "INSERT INTO posts(uuid, type, title, description, blood_group, coordinates, radius, location) VALUES($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography, $8, $9::jsonb) RETURNING post_uuid, created_at;"

	return query[section]
}

func getPostAppArgs(uuid, section string, post models.Post) ([]any, error) {
	args := make(map[string][]any, 0)

	if post.Location == nil {
		post.Location = []models.LocationObj{}
	}

	if post.Media == nil {
		post.Media = []string{}
	}

	locJSON, err := json.Marshal(post.Location)
	if err != nil {
		return nil, err
	}

	args["helpnearby"] = []any{uuid, section, post.Title, post.Description, post.Longitude, post.Latitude, post.Radius, locJSON}
	args["impactevents"] = []any{uuid, section, post.Title, post.Description, pq.Array(post.Media), post.Longitude, post.Latitude, post.EventAt, post.Radius, locJSON}
	args["moments"] = []any{uuid, section, post.Title, post.Description, pq.Array(post.Media)}
	args["missingpeople"] = []any{uuid, section, post.Title, post.Description, post.Gender, post.Age, pq.Array(post.Media), post.Longitude, post.Latitude, post.Radius, locJSON}
	args["bloodemergency"] = []any{uuid, section, post.Title, post.Description, post.BloodGroup, post.Longitude, post.Latitude, post.Radius, locJSON}

	return args[section], nil
}

func getGetAppQuery(section string) string {
	query := make(map[string]string, 0)

	query["helpnearby"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.created_at, p.location, u.profile_photo_url, EXISTS (SELECT 1 FROM interested i WHERE i.uuid = $2::text AND i.post_uuid = p.post_uuid) AS interested, u.name, u.phone, p.interested_count, p.comment_count FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND ST_DWithin(p.coordinates, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, p.radius) AND p.created_at >= $5 AND p.done = false ORDER BY p.created_at DESC;"
	query["impactevents"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, p.media, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.created_at, p.event_at, p.location, u.profile_photo_url, EXISTS (SELECT 1 FROM interested i WHERE i.uuid = $2::text AND i.post_uuid = p.post_uuid) AS interested, u.name, u.phone, p.interested_count, p.comment_count FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND ST_DWithin(p.coordinates, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, p.radius) AND p.event_at >= $5 AND p.done = false ORDER BY p.created_at DESC;"
	query["moments"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, p.media, p.created_at, p.location, u.profile_photo_url, EXISTS (SELECT 1 FROM interested i WHERE i.uuid = $2::text AND i.post_uuid = p.post_uuid) AS interested, u.name, p.interested_count, p.comment_count FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 ORDER BY p.created_at DESC LIMIT $3 OFFSET $4;"
	query["missingpeople"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, p.media, p.gender, p.age, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.created_at, p.location, u.profile_photo_url, EXISTS (SELECT 1 FROM interested i WHERE i.uuid = $2::text AND i.post_uuid = p.post_uuid) AS interested, u.name, u.phone, p.interested_count, p.comment_count FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND ST_DWithin(p.coordinates, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, p.radius) AND p.done = false ORDER BY p.created_at DESC LIMIT $5 OFFSET $6;"
	query["bloodemergency"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, p.blood_group, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.created_at, p.location, u.profile_photo_url, EXISTS (SELECT 1 FROM interested i WHERE i.uuid = $2::text AND i.post_uuid = p.post_uuid) AS interested, u.name, u.phone, p.interested_count, p.comment_count FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND ST_DWithin(p.coordinates, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, p.radius) AND p.created_at >= $5 AND p.done = false ORDER BY p.created_at DESC;"

	return query[section]
}

func getGetAppArgs(uuid, section string, page int, coordinates models.Coordinates) []any {
	args := make(map[string][]any, 0)

	var cutoff time.Time
	switch section {
	case "moments", "missingpeople":

	case "bloodemergency":
		cutoff = time.Now().UTC().Add(-24 * 3 * time.Hour)
	default:
		cutoff = time.Now().UTC().Add(-30 * time.Minute)
	}
	now := time.Now().UTC()

	limit := 20

	offset := (page - 1) * limit

	args["helpnearby"] = []any{section, uuid, coordinates.Longitude, coordinates.Latitude, cutoff}
	args["impactevents"] = []any{section, uuid, coordinates.Longitude, coordinates.Latitude, now}
	args["moments"] = []any{section, uuid, limit, offset}
	args["missingpeople"] = []any{section, uuid, coordinates.Longitude, coordinates.Latitude, limit, offset}
	args["bloodemergency"] = []any{section, uuid, coordinates.Longitude, coordinates.Latitude, cutoff}

	return args[section]
}

func getGetAppScan(section string, post *models.Post, locJSON *[]byte) []any {
	args := make(map[string][]any, 0)

	if post.Media == nil {
		post.Media = []string{}
	}

	args["helpnearby"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.Longitude, &post.Latitude, &post.CreatedAt, locJSON, &post.ProfilePhotoURL, &post.Interested, &post.Name, &post.Phone, &post.InterestedCount, &post.CommentCount}
	args["impactevents"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, pq.Array(&post.Media), &post.Longitude, &post.Latitude, &post.CreatedAt, &post.EventAt, locJSON, &post.ProfilePhotoURL, &post.Interested, &post.Name, &post.Phone, &post.InterestedCount, &post.CommentCount}
	args["moments"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, pq.Array(&post.Media), &post.CreatedAt, locJSON, &post.ProfilePhotoURL, &post.Interested, &post.Name, &post.InterestedCount, &post.CommentCount}
	args["missingpeople"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, pq.Array(&post.Media), &post.Gender, &post.Age, &post.Longitude, &post.Latitude, &post.CreatedAt, locJSON, &post.ProfilePhotoURL, &post.Interested, &post.Name, &post.Phone, &post.InterestedCount, &post.CommentCount}
	args["bloodemergency"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.BloodGroup, &post.Longitude, &post.Latitude, &post.CreatedAt, locJSON, &post.ProfilePhotoURL, &post.Interested, &post.Name, &post.Phone, &post.InterestedCount, &post.CommentCount}

	return args[section]
}

func getGetMyQuery(section string) string {
	query := make(map[string]string, 0)

	query["helpnearby"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.created_at, p.location, u.profile_photo_url, EXISTS (SELECT 1 FROM interested i WHERE i.uuid = $2::text AND i.post_uuid = p.post_uuid) AS interested, u.name, u.phone, p.interested_count, p.done, p.comment_count FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND p.uuid = $2 ORDER BY p.created_at DESC LIMIT $3 OFFSET $4;"
	query["impactevents"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, p.media, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.created_at, p.event_at, p.location, u.profile_photo_url, EXISTS (SELECT 1 FROM interested i WHERE i.uuid = $2::text AND i.post_uuid = p.post_uuid) AS interested, u.name, u.phone, p.interested_count, p.comment_count FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND p.uuid = $2 ORDER BY p.created_at DESC LIMIT $3 OFFSET $4;"
	query["moments"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, p.media, p.created_at, p.location, u.profile_photo_url, EXISTS (SELECT 1 FROM interested i WHERE i.uuid = $2::text AND i.post_uuid = p.post_uuid) AS interested, u.name, p.interested_count, p.comment_count FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND p.uuid = $2 ORDER BY p.created_at DESC LIMIT $3 OFFSET $4;"
	query["missingpeople"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, p.media, p.gender, p.age, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.created_at, p.location, u.profile_photo_url, EXISTS (SELECT 1 FROM interested i WHERE i.uuid = $2::text AND i.post_uuid = p.post_uuid) AS interested, u.name, u.phone, p.interested_count, p.done, p.comment_count FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND p.uuid = $2 ORDER BY p.created_at DESC LIMIT $3 OFFSET $4;"
	query["bloodemergency"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, p.blood_group, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.created_at, p.location, u.profile_photo_url, EXISTS (SELECT 1 FROM interested i WHERE i.uuid = $2::text AND i.post_uuid = p.post_uuid) AS interested, u.name, u.phone, p.interested_count, p.done, p.comment_count FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND p.uuid = $2 ORDER BY p.created_at DESC LIMIT $3 OFFSET $4;"

	return query[section]
}

func getGetMyArgs(uuid, section string, page int) []any {
	args := make(map[string][]any, 0)

	limit := 20

	offset := (page - 1) * limit

	args["helpnearby"] = []any{section, uuid, limit, offset}
	args["impactevents"] = []any{section, uuid, limit, offset}
	args["moments"] = []any{section, uuid, limit, offset}
	args["missingpeople"] = []any{section, uuid, limit, offset}
	args["bloodemergency"] = []any{section, uuid, limit, offset}

	return args[section]
}

func getGetMyScan(section string, post *models.Post, locJSON *[]byte) []any {
	args := make(map[string][]any, 0)

	if post.Media == nil {
		post.Media = []string{}
	}

	args["helpnearby"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.Longitude, &post.Latitude, &post.CreatedAt, locJSON, &post.ProfilePhotoURL, &post.Interested, &post.Name, &post.Phone, &post.InterestedCount, &post.Done, &post.CommentCount}
	args["impactevents"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, pq.Array(&post.Media), &post.Longitude, &post.Latitude, &post.CreatedAt, &post.EventAt, locJSON, &post.ProfilePhotoURL, &post.Interested, &post.Name, &post.Phone, &post.InterestedCount, &post.CommentCount}
	args["moments"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, pq.Array(&post.Media), &post.CreatedAt, locJSON, &post.ProfilePhotoURL, &post.Interested, &post.Name, &post.InterestedCount, &post.CommentCount}
	args["missingpeople"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, pq.Array(&post.Media), &post.Gender, &post.Age, &post.Longitude, &post.Latitude, &post.CreatedAt, locJSON, &post.ProfilePhotoURL, &post.Interested, &post.Name, &post.Phone, &post.InterestedCount, &post.Done, &post.CommentCount}
	args["bloodemergency"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.BloodGroup, &post.Longitude, &post.Latitude, &post.CreatedAt, locJSON, &post.ProfilePhotoURL, &post.Interested, &post.Name, &post.Phone, &post.InterestedCount, &post.Done, &post.CommentCount}

	return args[section]
}
