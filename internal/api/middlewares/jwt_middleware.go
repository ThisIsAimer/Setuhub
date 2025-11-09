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
			myErr := utils.ErrorHandler(err, "unauthorised", http.StatusBadRequest)
			utils.WriteJSONError(w, myErr.MyError.Error(), http.StatusBadRequest)
			return
		}

		jwtSecret := os.Getenv("JWT_SECRET")

		parsedToken, err := jwt.Parse(token.Value, func(token *jwt.Token) (any, error) {
			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return []byte(jwtSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				myErr := utils.ErrorHandler(err, "token expired", http.StatusUnauthorized)
				utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
				return
			}
			myErr := utils.ErrorHandler(err, "unauthorised access", http.StatusUnauthorized)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}

		if !parsedToken.Valid {
			utils.WriteJSONError(w, "invalid token", http.StatusUnauthorized)
			log.Println("invalid jwt:", token.Value)
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)

		if !ok {
			myErr := utils.ErrorHandler(err, "unauthorised token", http.StatusUnauthorized)
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
				unauthorized("email not verified")
				return
			}

		case "mail":
			if path != "/authenticate" {
				unauthorized("email verified but not authenticated")
				return
			}

		case "reset":
			if path != "/login/forgotpassword/otp" {
				unauthorized("please reset password first")
				return
			}

		default:
			// block unknown/missing auth states
			unauthorized("invalid authentication state")
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
