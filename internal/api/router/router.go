package router

import (
	"hackathon/internal/api/handlers"
	"net/http"
)

func Router() *http.ServeMux {
	mux := http.NewServeMux()

	//test----------------------------------------------------------------------------------------
	mux.HandleFunc("GET /home", handlers.Home)

	// auth crud-----------------------------------------------------------------------------------
	mux.HandleFunc("POST /signup", handlers.SignUpHandlerfunc)
	mux.HandleFunc("POST /signup/otp", handlers.SignUpOtpfunc)
	mux.HandleFunc("POST /authenticate", handlers.AuthenticationHandler)

	mux.HandleFunc("POST /login", handlers.LoginHandlerFunc)
	mux.HandleFunc("POST /logout", handlers.LogoutHandler)

	mux.HandleFunc("POST /login/forgotpassword", handlers.ForgotPassHandler)
	mux.HandleFunc("POST /login/forgotpassword/otp", handlers.ResetPassHandler)

	mux.HandleFunc("GET /profile", handlers.ViewProfile)

	// get location---------------------------------------------------------------------------------
	mux.HandleFunc("POST /coordinates", handlers.UpdateCoordinatesHandlerFunc)

	// app functions--------------------------------------------------------------------------------
	mux.HandleFunc("POST /help", handlers.HelpRequestPost)
	mux.HandleFunc("GET /help", handlers.HelpRequestGet)
	mux.HandleFunc("POST /event", handlers.EventRequestPost)
	mux.HandleFunc("GET /event", handlers.EventRequestGet)
	mux.HandleFunc("POST /post", handlers.MediaRequestPost)
	mux.HandleFunc("GET /post", handlers.MediaRequestGet)
	mux.HandleFunc("POST /missing", handlers.MissingRequestPost)
	mux.HandleFunc("GET /missing", handlers.MissingRequestGet)
	mux.HandleFunc("POST /blood", handlers.BloodRequestPost)
	mux.HandleFunc("GET /blood", handlers.BloodRequestGet)

	return mux
}
