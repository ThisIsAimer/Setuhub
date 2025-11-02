package databasehandler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hackathon/internal/models"
	"hackathon/pkg/utils"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func randOTP(n int) string {
	var letters = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func sendOTP(to, otp string) error {
	fromEmail := strings.TrimSpace(os.Getenv("FROM_EMAIL"))
	fromName := strings.TrimSpace(os.Getenv("FROM_NAME"))
	apiKey := strings.TrimSpace(os.Getenv("SENDGRID_API_KEY"))

	if fromEmail == "" || apiKey == "" {
		return utils.ErrorHandler(errors.New("missing FROM_EMAIL or SENDGRID_API_KEY"), "failed to send otp")
	}
	if to == "" || otp == "" {
		return utils.ErrorHandler(errors.New("missing recipient or otp"), "failed to send otp")
	}

	from := mail.NewEmail(fromName, fromEmail)
	toE := mail.NewEmail("", to)

	subject := "Your OTP code"
	plain := fmt.Sprintf(
		"Your OTP is: %s\nPlease enter the otp in our app\nOTP expires in 7 mins\n\n%s",
		otp, fromName,
	)

	msg := mail.NewSingleEmail(from, subject, toE, plain, "")

	// Disable tracking for OTPs (avoids link rewriting/spam signals).
	tracking := mail.NewTrackingSettings()

	click := mail.NewClickTrackingSetting()
	click.SetEnable(false)
	click.SetEnableText(false)
	tracking.SetClickTracking(click)

	open := mail.NewOpenTrackingSetting()
	open.SetEnable(false)
	tracking.SetOpenTracking(open)

	msg.SetTrackingSettings(tracking)

	// Optional: set a Reply-To if you have a support inbox
	// msg.SetReplyTo(mail.NewEmail("Support", "support@yourdomain.com"))

	client := sendgrid.NewSendClient(apiKey)
	resp, err := client.Send(msg)
	if err != nil {
		return fmt.Errorf("sendgrid send failed: %w", err)
	}
	if resp.StatusCode >= 400 {
		return utils.ErrorHandler(fmt.Errorf("sendgrid send failed: status=%d, body=%s", resp.StatusCode, resp.Body), "failed to send otp")
	}
	return nil
}

func getNameAndPhone(db *sql.DB, uuid string) (string, string, error) {

	var name string
	var number string

	err := db.QueryRow("SELECT name, phone FROM users WHERE uuid = $1", uuid).Scan(&name, &number)

	if err != nil {
		return "", "", utils.ErrorHandler(err, "error retrieveing name and number from database")
	}

	return name, number, nil

}

func getPostAppQuery(section string) string {
	query := make(map[string]string, 0)

	query["helpnearby"] = "INSERT INTO posts(uuid, type, title, description, coordinates, radius, location) VALUES($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography, $7, $8::jsonb) RETURNING post_uuid, created_at;"
	query["impactevents"] = "INSERT INTO posts(uuid, type, title, description,  media, coordinates, event_at, radius, location) VALUES($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography, $8, $9, $10::jsonb) RETURNING post_uuid, created_at, event_at;"
	query["moments"] = "INSERT INTO posts(uuid, type, title, description, media, coordinates, radius) VALUES($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography, $8) RETURNING post_uuid, created_at;"
	query["missingpeople"] = "INSERT INTO posts(uuid, type, title, description, gender, age,  media, coordinates, radius, location) VALUES($1, $2, $3, $4, $5, $6, $7, ST_SetSRID(ST_MakePoint($8, $9), 4326)::geography, $10, $11::jsonb) RETURNING post_uuid, created_at;"
	query["bloodemergency"] = "INSERT INTO posts(uuid, type, title, description, blood_group, coordinates, radius, location) VALUES($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography, $8, $9::jsonb) RETURNING post_uuid, created_at;"

	return query[section]
}

func getPostAppArgs(uuid, section string, post models.Post) ([]any, error) {
	args := make(map[string][]any, 0)

	if post.Location == nil {
		post.Location = []models.LocationObj{}
	}

	locJSON, err := json.Marshal(post.Location)
	if err != nil {
		return nil, utils.ErrorHandler(err, "error parsing location")
	}

	args["helpnearby"] = []any{uuid, section, post.Title, post.Description, post.Longitude, post.Latitude, post.Radius, locJSON}
	args["impactevents"] = []any{uuid, section, post.Title, post.Description, post.Media, post.Longitude, post.Latitude, post.EventAt, post.Radius, locJSON}
	args["moments"] = []any{uuid, section, post.Title, post.Description, post.Media, post.Longitude, post.Latitude, post.Radius}
	args["missingpeople"] = []any{uuid, section, post.Title, post.Description, post.Gender, post.Age, post.Media, post.Longitude, post.Latitude, post.Radius, locJSON}
	args["bloodemergency"] = []any{uuid, section, post.Title, post.Description, post.BloodGroup, post.Longitude, post.Latitude, post.Radius, locJSON}

	return args[section], nil
}

func getGetAppQuery(section string) string {
	query := make(map[string]string, 0)

	query["helpnearby"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.location, u.profile_photo_url, u.name, u.phone FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND ST_DWithin(p.coordinates, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, p.radius) AND p.created_at >= $4 AND p.done = false;"
	query["impactevents"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, p.media, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.event_at, p.location, u.profile_photo_url, u.name, u.phone FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND ST_DWithin(p.coordinates, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, p.radius) AND p.event_at >= $4 AND p.done = false;"
	query["moments"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.media, p.created_at, u.profile_photo_url, u.name, u.phone FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND ST_DWithin(p.coordinates, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, p.radius) AND p.created_at >= $4 AND p.done = false;"
	query["missingpeople"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, p.media, p.gender, p.age, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.location, u.profile_photo_url, u.name, u.phone FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND ST_DWithin(p.coordinates, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, p.radius) AND p.created_at >= $4 AND p.done = false;"
	query["bloodemergency"] = "SELECT p.post_uuid, p.uuid, p.title, p.description, p.blood_group, ST_X(p.coordinates::geometry) AS longitude, ST_Y(p.coordinates::geometry) AS latitude, p.location, u.profile_photo_url, u.name, u.phone FROM posts AS p JOIN users AS u ON u.uuid = p.uuid WHERE p.type = $1 AND ST_DWithin(p.coordinates, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, p.radius) AND p.created_at >= $4 AND p.done = false;"

	return query[section]
}

func getGetAppArgs(section string, coordinates models.Coordinates) []any {
	args := make(map[string][]any, 0)

	var cutoff time.Time
	if section == "moments" {
		cutoff = time.Now().UTC().Add(-240 * time.Hour)
	} else {
		cutoff = time.Now().UTC().Add(-30 * time.Minute)
	}
	now := time.Now().UTC() // renamed from `time`

	args["helpnearby"] = []any{section, coordinates.Longitude, coordinates.Latitude, cutoff}
	args["impactevents"] = []any{section, coordinates.Longitude, coordinates.Latitude, now}
	args["moments"] = []any{section, coordinates.Longitude, coordinates.Latitude, cutoff}
	args["missingpeople"] = []any{section, coordinates.Longitude, coordinates.Latitude, cutoff}
	args["bloodemergency"] = []any{section, coordinates.Longitude, coordinates.Latitude, cutoff}

	return args[section]
}

func getGetAppScan(section string, post *models.Post, locJSON *[]byte) []any {
	args := make(map[string][]any, 0)

	args["helpnearby"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.Longitude, &post.Latitude, locJSON, &post.ProfilePhotoURL, &post.Name, &post.Phone}
	args["impactevents"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.Media, &post.Longitude, &post.Latitude, &post.EventAt, locJSON, &post.ProfilePhotoURL, &post.Name, &post.Phone}
	args["moments"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.Longitude, &post.Latitude, &post.Media, &post.CreatedAt, &post.ProfilePhotoURL, &post.Name, &post.Phone}
	args["missingpeople"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.Media, &post.Gender, &post.Age, &post.Longitude, &post.Latitude, locJSON, &post.ProfilePhotoURL, &post.Name, &post.Phone}
	args["bloodemergency"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.BloodGroup, &post.Longitude, &post.Latitude, locJSON, &post.ProfilePhotoURL, &post.Name, &post.Phone}

	return args[section]
}
