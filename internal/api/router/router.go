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
	mux.HandleFunc("PATCH /update/photo", handlers.UpdateProfilePhoto)
	mux.HandleFunc("PATCH /update/phonenumber", handlers.UpdatePhoneNumber)

	// get location---------------------------------------------------------------------------------
	mux.HandleFunc("POST /coordinates", handlers.UpdateCoordinatesHandlerFunc)

	// create button ---------------------------------------------------------------------------
	mux.HandleFunc("GET /request/create", handlers.CreatePostButtonHandler)
	// app functions--------------------------------------------------------------------------------
	mux.HandleFunc("POST /request/create/{section}", handlers.HandleRequestCreate)
	mux.HandleFunc("GET /request/retrieve/{section}", handlers.HandleRequestRetrieve)
	mux.HandleFunc("GET /request/my/{section}", handlers.HandleMyRequestRetrieve)
	mux.HandleFunc("PATCH /request/done/{postid}", handlers.HandleRequestDone)
	// --- expo--------------------------------------------------------------------
	mux.HandleFunc("POST /set/token", handlers.SetFirebaseToken)
	// --- interested ------------------------------------------------------------
	mux.HandleFunc("PATCH /request/interested/{postuuid}", handlers.InterestedPostHandler)
	mux.HandleFunc("PATCH /request/uninterested/{postuuid}", handlers.UninterestedPostHandler)

	// --- comments ---------------------------------------------------------------------
	mux.HandleFunc("GET /request/comments/{postuuid}", handlers.GetCommentHandler)
	mux.HandleFunc("POST /request/comment/create", handlers.CreateCommentHandler)
	mux.HandleFunc("PATCH /request/comment/edit/{commentuuid}", handlers.EditCommentHandler)
	mux.HandleFunc("DELETE /request/comment/delete/{commentuuid}", handlers.DeleteCommentHandler)

	return mux
}
