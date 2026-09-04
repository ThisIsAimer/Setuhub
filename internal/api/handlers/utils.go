package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
)

func verifyDiditSignature(body []byte, signature string) bool {
	secret := os.Getenv("DIDIT_WEBHOOK_SECRET")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)

	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal(
		[]byte(expected),
		[]byte(signature),
	)
}
