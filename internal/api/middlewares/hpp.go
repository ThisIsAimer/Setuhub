package middlewares

import (
	"fmt"
	"net/http"
	"strings"
)

type HppOptions struct {
	CheckQuery              bool
	CheckBody               bool
	CheckBodyForContentType string
	WhiteList               []string
}

func Hpp(options HppOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if options.CheckBody && r.Method == http.MethodPost && isCorrectContentType(r, options.CheckBodyForContentType) {
				filterBodyParams(r, options.WhiteList)
			}
			if options.CheckQuery && r.URL.Query() != nil {
				filterQueryParams(r, options.WhiteList)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isCorrectContentType(r *http.Request, contentType string) bool {
	return strings.Contains(r.Header.Get("Content-Type"), contentType)
}

func filterQueryParams(r *http.Request, whitelist []string) {

	query := r.URL.Query()

	for k, v := range query {
		if len(v) > 1 {
			query.Set(k, v[0]) //first value
			// query.Set(k,v[len(v)-1]) //last value
		}

		if !isWhiteListerd(k, whitelist) {
			query.Del(k)
		}
	}

	r.URL.RawQuery = query.Encode()
}

func filterBodyParams(r *http.Request, whitelist []string) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println("error is:", err)
		return
	}

	for k, v := range r.PostForm {
		if len(v) > 1 {
			r.PostForm.Set(k, v[0]) //first value
			// r.PostForm.Set(k, v[len(v)-1]) //last value
		}
		if !isWhiteListerd(k, whitelist) {
			delete(r.PostForm, k)
		}
	}
}

func isWhiteListerd(param string, whitelist []string) bool {
	for _, v := range whitelist {
		if param == v {
			return true
		}
	}
	return false
}
