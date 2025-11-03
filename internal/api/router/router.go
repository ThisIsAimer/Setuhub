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
	mux.HandleFunc("POST /profile/photo", handlers.UpdateProfilePhoto)

	// get location---------------------------------------------------------------------------------
	mux.HandleFunc("POST /coordinates", handlers.UpdateCoordinatesHandlerFunc)

	// app functions--------------------------------------------------------------------------------
	mux.HandleFunc("POST /request/create/{section}", handlers.HandleRequestCreate)
	mux.HandleFunc("GET /request/retrieve/{section}", handlers.HandleRequestRetrieve)
	mux.HandleFunc("PATCH /request/done/{postid}", handlers.HandleRequestDone)
	// --- interested ------------------------------------------------------------
	mux.HandleFunc("PATCH /request/interested/{postuuid}", handlers.InterestedPostHandler)
	mux.HandleFunc("PATCH /request/uninterested/{postuuid}", handlers.UninterestedPostHandler)

	return mux
}
