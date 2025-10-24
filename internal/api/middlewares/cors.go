package middlewares

import (
	"net/http"
	"os"
	"strings"
)

// cross-origine resource sharing
func Cors(next http.Handler) http.Handler {

	// getting origins from env
	var allowedOrigins = strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if originVerification(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			http.Error(w, "not allowed by Cors", http.StatusForbidden)
			return
		}

		// Set other CORS headers
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// method options is for a preflight check
		//A preflight check refers to a preliminary request made by browsers when using CORS (Cross-Origin Resource Sharing) to ensure that the actual request is safe to send.
		if r.Method == http.MethodOptions {
			return
		}

		next.ServeHTTP(w, r)
	})

}

func originVerification(origin string, allowedOrigins []string) bool {
	for _, value := range allowedOrigins {
		if value == origin {
			return true
		}
	}
	return false
}