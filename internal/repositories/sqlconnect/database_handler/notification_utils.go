package databasehandler

import (
	"context"
	"fmt"
	"hackathon/internal/models"
	"hackathon/internal/repositories/sqlconnect"
	"hackathon/pkg/utils"
	"log"
	"os"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// firebase cloud messaging--------------------------------------------------------------------
func newFCM() (*messaging.Client, error) {
	ctx := context.Background()
	// Either rely on GOOGLE_APPLICATION_CREDENTIALS or pass the file:
	credPath := os.Getenv("GOOGLE_SERVICE_CREDENTIALS")
	if credPath == "" {
		// fallback for local dev if env missing
		credPath = "cmd/api/google-services.json"
	}
	opt := option.WithCredentialsFile(credPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, err
	}
	return app.Messaging(ctx)
}

// sending tokens --------------------------------------------------------------------------------------------------------------------------------------------------

func sendToOne(ctx context.Context, client *messaging.Client, token string) error {

	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("empty FCM token")
	}

	message := &messaging.Message{
		Token: token, // single device token string
		Notification: &messaging.Notification{
			Title: "someone has shown interest in your request!",
			Body:  "A user on the way",
		},
		Data: map[string]string{
			"link": "https://yourapp.com/",
			"type": "information",
		},
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		return err
	}

	log.Printf("Successfully sent message: %s", response)
	return nil
}

func sendToMany(ctx context.Context, client *messaging.Client, tokens []string, post models.Post) (string, error) {

	message := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: fmt.Sprintf("New %s request: %s", post.Type, post.Title),
			Body:  post.Description,
		},
		Data: map[string]string{
			"link": fmt.Sprintf("https://yourapp.com/%s", post.PostUUID),
			"type": post.Type,
		},
	}

	results, err := client.SendEachForMulticast(ctx, message)
	if err != nil {
		return "", fmt.Errorf("failed to send multicast message: %w", err)
	}

	var (
		successCount  int
		failCount     int
		invalidTokens []string
	)

	for i, resp := range results.Responses {
		if resp.Success {
			successCount++
		} else {
			failCount++
			// Handle unregistered/invalid tokens
			if messaging.IsUnregistered(resp.Error) {
				invalidTokens = append(invalidTokens, tokens[i])
			}
		}
	}

	fmt.Printf("Summary: %d succeeded, %d failed\n", successCount, failCount)
	if len(invalidTokens) > 0 {
		fmt.Printf("Found %d invalid tokens, should remove: %v\n", len(invalidTokens), invalidTokens)
	}

	return fmt.Sprintf("successfully notified %d people", successCount), nil
}

// sending notis--------------------------------------------------------------------------------------------------------------------------------------------------
func sendNotification(uuid string) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		log.Println(err, "unable to notify")
		return
	}
	defer db.Close()

	var firebaseToken string

	query := `SELECT firebase_token FROM users WHERE uuid = $1;`

	err = db.QueryRow(query, uuid).Scan(&firebaseToken)
	if err != nil {
		log.Println(err, "unable to notify")
		return
	}

	fcmConn, err := newFCM()
	if err != nil {
		log.Println(err, "unable to notify")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = sendToOne(ctx, fcmConn, firebaseToken)
	if err != nil {
		return
	}

}
func sendNotifications(post models.Post, noti chan string) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		myErr := utils.ErrorHandler(err, "unable to notify", 0)
		noti <- myErr.MyError.Error()
		return
	}
	defer db.Close()

	query := `SELECT firebase_token FROM users WHERE ST_DWithin(coordinates, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3);`

	rows, err := db.Query(query,
		post.Longitude, post.Latitude, post.Radius,
	)
	if err != nil {
		myErr := utils.ErrorHandler(err, "unable to notify", 0)
		noti <- myErr.MyError.Error()
		return
	}
	defer rows.Close()

	tokens := make([]string, 0)

	for rows.Next() {
		var token string

		err := rows.Scan(&token)

		if err != nil {
			myErr := utils.ErrorHandler(err, "unable to notify", 0)
			noti <- myErr.MyError.Error()
			return
		}

		if token != "" {
			tokens = append(tokens, token)
		}

	}

	if len(tokens) == 0 {
		noti <- "No people in area to notify"
		return
	}

	fcmConn, err := newFCM()
	if err != nil {
		myErr := utils.ErrorHandler(err, "unable to notify", 0)
		noti <- myErr.MyError.Error()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notified, err := sendToMany(ctx, fcmConn, tokens, post)
	if err != nil {
		myErr := utils.ErrorHandler(err, "unable to notify", 0)
		noti <- myErr.MyError.Error()
		return
	}

	noti <- notified

}
