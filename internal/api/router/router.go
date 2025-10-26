package router

import (
	"hackathon/internal/api/handlers"
	"net/http"
)

func Router() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /signup", handlers.SignUpHandlerfunc)
	mux.HandleFunc("POST /signup/otp", handlers.SignUpOtpfunc)
	mux.HandleFunc("POST /authenticate", handlers.AuthenticationHandler)

	mux.HandleFunc("POST /login", handlers.LoginHandlerFunc)
	mux.HandleFunc("POST /logout", handlers.LogoutHandler)

	mux.HandleFunc("POST /login/forgotpassword", handlers.ForgotPassHandler)
	mux.HandleFunc("POST /login/forgotpassword/otp", handlers.ResetPassHandler)

	mux.HandleFunc("GET /home", handlers.Home)

	return mux
}
