package databasehandler

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"

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
		return errors.New("missing FROM_EMAIL or SENDGRID_API_KEY")
	}
	if to == "" || otp == "" {
		return errors.New("missing recipient or otp")
	}

	from := mail.NewEmail(fromName, fromEmail)
	toE := mail.NewEmail("", to)

	subject := "Your OTP code"
	plain := fmt.Sprintf(
		"Your OTP is: %s\nPlease enter thin in our app\n\n%s",
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
		return fmt.Errorf("sendgrid send failed: status=%d, body=%s", resp.StatusCode, resp.Body)
	}
	return nil
}
