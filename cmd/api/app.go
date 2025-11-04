package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	mid "hackathon/internal/api/middlewares"
	"hackathon/internal/api/router"
	"hackathon/pkg/utils"

	_ "github.com/joho/godotenv"
)

func main() {

	var err error

	// err = godotenv.Load(`cmd\api\.env`)
	// if err != nil {
	// 	fmt.Println("Failed to load env")
	// 	return
	// }

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	} // local fallback

	rateLimiter := mid.NewRateLimiter(5, time.Second*5)

	whiteList := []string{
		"latitude",
		"longitude",
	}

	hppSettings := &mid.HppOptions{
		CheckQuery:              true,
		CheckBody:               true,
		CheckBodyForContentType: "application/x-www-form-urlencoded",
		WhiteList:               whiteList,
	}

	hppMiddleware := mid.Hpp(*hppSettings)

	jwtMiddleware := mid.SkipJwtRoutes(mid.JwtMiddleware, "/signup", "/login", "/logout", "/login/forgotpassword", "/healthz")

	secureRouter := utils.ApplyMiddlewares(router.Router(), mid.SecurityHeaders, jwtMiddleware, hppMiddleware, mid.XSSMiddleware, rateLimiter.Middleware, mid.Cors)

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
