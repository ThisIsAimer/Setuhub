package middlewares

import (
	"fmt"
	"hackathon/pkg/utils"
	"net/http"
	"os"

)

// cross-origine resource sharing
func Cors(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Set other CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}

		RealappSecret := os.Getenv("APP_SECRET")
		appSecret := r.Header.Get("X-App-Secret")

		if RealappSecret != appSecret {
			myErr := utils.ErrorHandler(fmt.Errorf(`wrong app Secret: "%s"`, appSecret), "not allowed by cors", http.StatusBadRequest)
			http.Error(w, myErr.MyError.Error(), myErr.Status)
			return
		}

		// method options is for a preflight check
		//A preflight check refers to a preliminary request made by browsers when using CORS (Cross-Origin Resource Sharing) to ensure that the actual request is safe to send.
		if r.Method == http.MethodOptions {
			return
		}

		next.ServeHTTP(w, r)
	})

}
