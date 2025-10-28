package databasehandler

import (
	"crypto/tls"
	"hackathon/pkg/utils"
	"math/rand"
	"os"

	mail "gopkg.in/gomail.v2"
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
	from := os.Getenv("EMAIL")
	appPass := os.Getenv("APP_PASSWORD") // <-- replace with your REAL 16-char Gmail App Password

	m := mail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "otp for our app")
	m.SetBody("text/plain", "your otp is:"+otp)

	d := mail.NewDialer("smtp.gmail.com", 465, from, appPass)
	d.SSL = true
	d.TLSConfig = &tls.Config{ServerName: "smtp.gmail.com"}

	if err := d.DialAndSend(m); err != nil {
		return utils.ErrorHandler(err, "error sending email")
	}

	return nil
}
