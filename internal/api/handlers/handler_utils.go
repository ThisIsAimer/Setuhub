package handlers

import (
	"context"
	"fmt"
	"hackathon/pkg/utils"
	"reflect"
	"regexp"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

func checkEmptyField(modle any) error {
	modleValue := reflect.ValueOf(modle)
	modleType := modleValue.Type()

	for i := range modleType.NumField() {
		dbTag := modleType.Field(i).Tag.Get("db")

		if modleValue.Field(i).String() == "" {
			return fmt.Errorf("empty fields found %s", dbTag)
		}

	}

	return nil
}

func isValidEmailFormat(email string) error {
	var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	if !emailRegex.MatchString(email) {
		return utils.ErrorHandler(fmt.Errorf("email format wrong"), "invalid email")
	}
	return nil
}

func checkSection(section string) error {
	switch section {
	case "help", "event", "media", "missing", "blood":
		return nil
	default:
		return utils.ErrorHandler(fmt.Errorf("invalid section: %s", section), "invalid route")
	}
}

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

func sendToMany(ctx context.Context, client *messaging.Client, tokens []string, request, title, description string) error {
	if len(tokens) == 0 {
		return fmt.Errorf("no FCM tokens provided")
	}

	message := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: fmt.Sprintf("New %s request: %s", request, title),
			Body:  description,
		},
		Data: map[string]string{
			"link": fmt.Sprintf("https://yourapp.com/%s", request),
			"type": request,
		},
	}

	results, err := client.SendEachForMulticast(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send multicast message: %w", err)
	}

	var (
		successCount int
		failCount    int
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

	// Cleanup logic for unregistered tokens
	if len(invalidTokens) > 0 {
		fmt.Printf("Found %d invalid tokens, should remove: %v\n", len(invalidTokens), invalidTokens)
		// Example: deleteInvalidTokensFromDB(invalidTokens)
	}

	return nil
}
