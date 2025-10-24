package middlewares

import (
	"net/http"
	"strings"
)

func SkipJwtRoutes(jwtMiddleware func(http.Handler) http.Handler, excludedPaths ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, paths := range excludedPaths {
				if strings.HasPrefix(r.URL.Path, paths) {
					next.ServeHTTP(w, r) // skipping jwt middleware
					return
				}
			}
			// the code only goes through jwt middleware of no paths match
			jwtMiddleware(next).ServeHTTP(w, r)
		})

	}
}