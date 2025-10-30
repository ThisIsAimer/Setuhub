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

	query["help"] = "INSERT INTO posts(uuid, type, title, description, coordinates, radius, location) VALUES($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography, $7, $8::jsonb);"
	query["event"] = "INSERT INTO posts(uuid, type, title, description, coordinates, event_at, radius, location) VALUES($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography, $7, $8, $9::jsonb);"
	query["media"] = "INSERT INTO posts(uuid, type, title, description, media, coordinates, radius) VALUES($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography, $8);"
	query["missing"] = "INSERT INTO posts(uuid, type, title, description, gender, age, coordinates, radius, location) VALUES($1, $2, $3, $4, $5, $6, ST_SetSRID(ST_MakePoint($7, $8), 4326)::geography, $9, $10::jsonb);"
	query["blood"] = "INSERT INTO posts(uuid, type, title, description, blood_group, coordinates, radius, location) VALUES($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography, $8, $9::jsonb);"

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

	args["help"] = []any{uuid, "help", post.Title, post.Description, post.Longitude, post.Latitude, post.Radius, locJSON}
	args["event"] = []any{uuid, "event", post.Title, post.Description, post.Longitude, post.Latitude, post.EventAt, post.Radius, locJSON}
	args["media"] = []any{uuid, "media", post.Title, post.Description, post.Media, post.Longitude, post.Latitude, post.Radius}
	args["missing"] = []any{uuid, "missing", post.Title, post.Description, post.Gender, post.Age, post.Longitude, post.Latitude, post.Radius, locJSON}
	args["blood"] = []any{uuid, "blood", post.Title, post.Description, post.BloodGroup, post.Longitude, post.Latitude, post.Radius, locJSON}

	return args[section], nil
}

func getGetAppQuery(section string) string {
	query := make(map[string]string, 0)

	query["help"] = "SELECT post_uuid, uuid, title, description, ST_X(coordinates::geometry) AS longitude, ST_Y(coordinates::geometry) AS latitude, location FROM posts WHERE type = $1 AND ST_DWithin(coordinates, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, radius) AND created_at >= $4;"
	query["event"] = "SELECT post_uuid, uuid, title, description, ST_X(coordinates::geometry) AS longitude, ST_Y(coordinates::geometry) AS latitude, event_at, location FROM posts WHERE type = $1 AND ST_DWithin(coordinates, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, radius) AND event_at >= $4;"
	query["media"] = "SELECT post_uuid, uuid, title, description, ST_X(coordinates::geometry) AS longitude, ST_Y(coordinates::geometry) AS latitude, media, created_at FROM posts WHERE type = $1 AND ST_DWithin(coordinates, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, radius) AND created_at >= $4;"
	query["missing"] = "SELECT post_uuid, uuid, title, description, gender, age, ST_X(coordinates::geometry) AS longitude, ST_Y(coordinates::geometry) AS latitude, location FROM posts WHERE type = $1 AND ST_DWithin(coordinates, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, radius) AND created_at >= $4;"
	query["blood"] = "SELECT post_uuid, uuid, title, description, blood_group, ST_X(coordinates::geometry) AS longitude, ST_Y(coordinates::geometry) AS latitude, location FROM posts WHERE type = $1 AND ST_DWithin(coordinates, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, radius) AND created_at >= $4;"

	return query[section]
}

func getGetAppArgs(section string, coordinates models.Coordinates) []any {
	args := make(map[string][]any, 0)

	var cutoff time.Time
	if section == "media" {
		cutoff = time.Now().UTC().Add(-240 * time.Hour)
	} else {
		cutoff = time.Now().UTC().Add(-30 * time.Minute)
	}
	now := time.Now().UTC() // renamed from `time`

	args["help"] = []any{"help", coordinates.Longitude, coordinates.Latitude, cutoff}
	args["event"] = []any{"event", coordinates.Longitude, coordinates.Latitude, now}
	args["media"] = []any{"media", coordinates.Longitude, coordinates.Latitude, cutoff}
	args["missing"] = []any{"missing", coordinates.Longitude, coordinates.Latitude, cutoff}
	args["blood"] = []any{"blood", coordinates.Longitude, coordinates.Latitude, cutoff}

	return args[section]
}

func getGetAppScan(section string, post *models.Post, locJSON *[]byte) []any {
	args := make(map[string][]any, 0)

	args["help"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.Longitude, &post.Latitude, locJSON}
	args["event"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.Longitude, &post.Latitude, &post.EventAt, locJSON}
	args["media"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.Longitude, &post.Latitude, &post.Media, &post.CreatedAt}
	args["missing"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.Gender, &post.Age, &post.Longitude, &post.Latitude, locJSON}
	args["blood"] = []any{&post.PostUUID, &post.UUID, &post.Title, &post.Description, &post.BloodGroup, &post.Longitude, &post.Latitude, locJSON}

	return args[section]
}
