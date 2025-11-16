package middlewares

import (
	"context"
	"errors"
	"fmt"
	"hackathon/pkg/utils"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

func JwtMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var targetUrl string

		var devEnv = r.Header.Get("X-App-Environment")

		switch devEnv {
		case "dev":
			targetUrl = "setuhub://"
		case "prod":
			targetUrl = "https://setuhub.io/"

		default:
			myErr := utils.ErrorHandler(fmt.Errorf(`invalid X-App-Environment:"%s"`, devEnv), "Invalid Environment details", http.StatusBadRequest)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return

		}

		//fmt.Println(r.Cookies())
		token, err := r.Cookie("Bearer")
		if err != nil {
			http.Redirect(w, r, targetUrl, http.StatusFound)
			return
		}

		jwtSecret := os.Getenv("JWT_SECRET")

		parsedToken, err := jwt.Parse(token.Value, func(token *jwt.Token) (any, error) {
			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return []byte(jwtSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				http.Redirect(w, r, targetUrl, http.StatusFound)
				return
			}
			http.Redirect(w, r, targetUrl, http.StatusFound)
			return
		}

		if !parsedToken.Valid {
			http.Redirect(w, r, targetUrl, http.StatusFound)
			return
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)

		if !ok {
			myErr := utils.ErrorHandler(err, "Unauthorised token", http.StatusUnauthorized)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}

		auth, _ := claims["auth"].(string)

		path := normalizePath(r.URL.Path)

		unauthorized := func(msg string) {
			myErr := utils.ErrorHandler(errors.New(msg), msg, http.StatusUnauthorized)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
		}

		switch auth {
		case "verified":
			// ok, proceed

		case "unverified":
			if path != "/signup/otp" {
				unauthorized("Email not verified")
				return
			}

		case "mail_verified":
			if path != "/authenticate" {
				unauthorized("Email verified but not authenticated")
				return
			}

		case "reset":
			if path != "/login/forgotpassword/otp" {
				unauthorized("Please reset password first")
				return
			}

		default:
			// block unknown/missing auth states
			unauthorized("Invalid authentication state")
			return
		}

		ctx := context.WithValue(r.Context(), utils.JwtKey("uuid"), claims["uuid"])
		ctx = context.WithValue(ctx, utils.JwtKey("role"), claims["role"])
		ctx = context.WithValue(ctx, utils.JwtKey("auth"), claims["auth"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

func normalizePath(path string) string {
	// optional: normalize trailing slash, etc.
	if len(path) > 1 && path[len(path)-1] == '/' {
		return path[:len(path)-1]
	}
	return path
}
