package databasehandler

import (
	"context"
	"fmt"
	"hackathon/internal/models"
	"hackathon/internal/repositories/sqlconnect"
	"hackathon/pkg/utils"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

func newFCM() (*messaging.Client, error) {
	ctx := context.Background()
	// Either rely on GOOGLE_APPLICATION_CREDENTIALS or pass the file:
	opt := option.WithCredentialsFile("service-account.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, err
	}
	return app.Messaging(ctx)
}

// --------------------------------------------------------------------------------------------------------------------------------------------------
func sendToMany(ctx context.Context, client *messaging.Client, tokens []string, post models.Post) (string, error) {
	if len(tokens) == 0 {
		return fmt.Sprintln("people in area to be notified provided"), nil
	}

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

//--------------------------------------------------------------------------------------------------------------------------------------------------

func sendNotifications(post models.Post, noti chan string) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		myErr := utils.ErrorHandler(err, "unable to notify")
		noti <- myErr.Error()
		return
	}
	defer db.Close()

	query := `SELECT firebase_token FROM users WHERE ST_DWithin(coordinates, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3);`

	rows, err := db.Query(query,
		post.Longitude, post.Latitude, post.Radius,
	)
	if err != nil {
		myErr := utils.ErrorHandler(err, "unable to notify")
		noti <- myErr.Error()
		return
	}
	defer rows.Close()

	tokens := make([]string, 0)

	for rows.Next() {
		var token string

		err := rows.Scan(&token)

		if err != nil {
			myErr := utils.ErrorHandler(err, "unable to notify")
			noti <- myErr.Error()
			return
		}

		if token != "" {
			tokens = append(tokens, token)
		}

	}

	fcmConn, err := newFCM()
	if err != nil {
		myErr := utils.ErrorHandler(err, "unable to notify")
		noti <- myErr.Error()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	notified, err := sendToMany(ctx, fcmConn, tokens, post)
	if err != nil {
		myErr := utils.ErrorHandler(err, "unable to notify")
		noti <- myErr.Error()
		return
	}

	noti <- notified

}
