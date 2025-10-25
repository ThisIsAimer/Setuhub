package middlewares

import (
	"context"
	"errors"
	"hackathon/pkg/utils"
	"log"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

func JwtMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//fmt.Println(r.Cookies())
		token, err := r.Cookie("Bearer")
		if err != nil {
			http.Error(w, "unauthorised", http.StatusBadRequest)
			return
		}

		jwtSecret := os.Getenv("JWT_SECRET")

		parsedToken, err := jwt.Parse(token.Value, func(token *jwt.Token) (any, error) {
			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return []byte(jwtSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				myErr := utils.ErrorHandler(err, "token expired")
				http.Error(w, myErr.Error(), http.StatusUnauthorized)
				return
			}
			myErr := utils.ErrorHandler(err, "unauthorised access")
			http.Error(w, myErr.Error(), http.StatusUnauthorized)
			return
		}

		if !parsedToken.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			log.Println("invalid jwt:", token.Value)
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)

		if !ok {
			myErr := utils.ErrorHandler(err, "unauthorised token")
			http.Error(w, myErr.Error(), http.StatusUnauthorized)
			return
		}

		if claims["auth"] == "unverified" {
			myErr := utils.ErrorHandler(err, "not verified")
			http.Error(w, myErr.Error(), http.StatusUnauthorized)
			return

		}

		if r.URL.Path != "/authenticate" {
			if claims["auth"] == "mail" {
				myErr := utils.ErrorHandler(err, "mail verified not authenticated")
				http.Error(w, myErr.Error(), http.StatusUnauthorized)
				return

			}
		}

		if r.URL.Path == "/authenticate"{
			if claims["auth"] == "verified" {
				myErr := utils.ErrorHandler(err, "user already verified")
				http.Error(w, myErr.Error(), http.StatusBadRequest)
				return

			}
		}

		ctx := context.WithValue(r.Context(), utils.JwtKey("uuid"), claims["uuid"])
		ctx = context.WithValue(ctx, utils.JwtKey("role"), claims["role"])
		ctx = context.WithValue(ctx, utils.JwtKey("auth"), claims["auth"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})

}
