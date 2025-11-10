package utils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Errorhandler struct {
	MyError error
	Status  int
}

func WriteJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"status": "Error", "message": message})
}

func ErrorHandler(err error, message string, status int) Errorhandler {
	errorLogger := log.New(os.Stderr, "ERROR:", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger.Println(message, ":-", err)

	shownError := errors.New(message)
	return Errorhandler{MyError: shownError, Status: status}
}

func PassEncoder(password string, salt []byte) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password is empty")
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	saltBase64 := base64.StdEncoding.EncodeToString(salt)
	hashBase64 := base64.StdEncoding.EncodeToString(hash)

	encodedHash := saltBase64 + "." + hashBase64

	return encodedHash, nil
}

func VerifyPassword(givenPass, realPass string) error {

	parts := strings.Split(realPass, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid encode hash format")
	}

	saltBase64 := parts[0]

	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return err
	}

	givenPass, err = PassEncoder(givenPass, salt)
	if err != nil {
		fmt.Println("error is:", err)
		return err
	}

	if givenPass != realPass {
		return fmt.Errorf("password doesnt match")
	}

	return nil
}
