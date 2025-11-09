package utils

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtKey string

func SignToken(userId, role, auth string) (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	jwtExpires := os.Getenv("JWT_EXPIRES_IN")

	claims := jwt.MapClaims{
		"uuid": userId,
		"role": role,
		"auth": auth,
	}

	if jwtExpires != "" {
		duration, err := time.ParseDuration(jwtExpires)
		if err != nil {
			log.Println(err)
			return "", fmt.Errorf("error parsing jwt duration")
		}
		claims["exp"] = time.Now().Add(duration).Unix()
	} else {
		claims["exp"] = time.Now().Add(15 * time.Minute).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(jwtSecret))

	if err != nil {
		log.Println(err)
		return "", fmt.Errorf("couldnt sign token")
	}

	return signedToken, nil
}
