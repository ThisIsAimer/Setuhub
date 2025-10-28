package databasehandler

import (
	"fmt"
	"math/rand"
	"os"

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
	fromEmail := os.Getenv("FROM_EMAIL")    // e.g. rudra@gmail.com (verified in SendGrid)
	fromName := os.Getenv("FROM_NAME")      // e.g. Our Hackathon App
	apiKey := os.Getenv("SENDGRID_API_KEY") // from SendGrid dashboard

	from := mail.NewEmail(fromName, fromEmail)
	toE := mail.NewEmail("", to)

	subject := "OTP for our app"
	text := "Your OTP is: " + otp

	msg := mail.NewSingleEmail(from, subject, toE, text, "")
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
