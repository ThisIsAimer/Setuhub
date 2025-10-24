package utils

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/argon2"
)

func ErrorHandler(err error, message string) error {
	errorLogger := log.New(os.Stderr, "ERROR:", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger.Println(message, ":-", err)

	return fmt.Errorf("%v", message)
}

func PassEncoder(password string, salt []byte) (string, error) {
	if password == "" {
		return "", ErrorHandler(fmt.Errorf("password is empty"), "password is required")
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
		return ErrorHandler(fmt.Errorf("invalid encode hash format"), "Password must be reset")
	}

	saltBase64 := parts[0]

	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return ErrorHandler(err, "error decoding salt")
	}

	givenPass, err = PassEncoder(givenPass, salt)
	if err != nil {
		fmt.Println("error is:", err)
		return ErrorHandler(err, "error encoding password")
	}

	if givenPass != realPass {
		return ErrorHandler(fmt.Errorf("password doesnt match"), "incorrect password")
	}

	return nil
}
