package router

import (
	"hackathon/internal/api/handlers"
	"net/http"
)

func Router() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /signup", handlers.SignUpHandlerfunc)
	mux.HandleFunc("POST /login", handlers.LoginHandlerFunc)
	mux.HandleFunc("POST /logout", handlers.LogoutHandler)

	mux.HandleFunc("POST /login/forgotpassword", handlers.ForgotPassHandler)
	mux.HandleFunc("POST /login/forgotpassword/reset/{resetcode}", handlers.ResetPassHandler)

	return mux
}
