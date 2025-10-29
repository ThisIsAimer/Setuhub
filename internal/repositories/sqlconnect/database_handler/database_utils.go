package databasehandler

import (
	"database/sql"
	"errors"
	"fmt"
	"hackathon/pkg/utils"
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
		return utils.ErrorHandler(errors.New("missing FROM_EMAIL or SENDGRID_API_KEY"), "failed to send otp")
	}
	if to == "" || otp == "" {
		return utils.ErrorHandler(errors.New("missing recipient or otp"),"failed to send otp")
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
		return utils.ErrorHandler(fmt.Errorf("sendgrid send failed: status=%d, body=%s", resp.StatusCode, resp.Body),"failed to send otp")
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
