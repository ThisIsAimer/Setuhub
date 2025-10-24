package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	mid "hackathon/internal/api/middlewares"
	"hackathon/internal/api/router"
	"hackathon/pkg/utils"

	"github.com/joho/godotenv"
)

func main() {

	var err error

	err = godotenv.Load(`cmd\api\.env`)
	if err != nil {
		fmt.Println("Failed to load env")
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	} // local fallback

	rateLimiter := mid.NewRateLimiter(5, time.Second*5)

	jwtMiddleware := mid.SkipJwtRoutes(mid.JwtMiddleware, "/login", "/signup", "/login/forgotpassword", "/login/forgotpassword/reset/")

	secureRouter := utils.ApplyMiddlewares(router.Router(), mid.SecurityHeaders, jwtMiddleware, mid.XSSMiddleware, rateLimiter.Middleware, mid.Cors)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: secureRouter,
	}

	fmt.Println("server is running on port", port)

	err = server.ListenAndServe()
	if err != nil {
		fmt.Println("error starting server")
		return
	}

}
