package utils

import "net/http"

type Middleware func(http.Handler) http.Handler

func ApplyMiddlewares(handler http.Handler, middlewares ...Middleware) http.Handler {

	for _, v := range middlewares {
		handler = v(handler)
	}
	return handler
}