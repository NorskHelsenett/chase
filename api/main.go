package main

import (
	"log"

	"git.torden.tech/jonasbg/fit/auth"
	"git.torden.tech/jonasbg/fit/database"
	"git.torden.tech/jonasbg/fit/handlers"
	"git.torden.tech/jonasbg/fit/security"
	"git.torden.tech/jonasbg/fit/session"
	"git.torden.tech/jonasbg/fit/spa"
	"git.torden.tech/jonasbg/fit/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var db *gorm.DB

func setupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/login", handlers.HandleLogin)
		api.GET("/callback", handlers.HandleCallback)
		api.GET("/logout", handlers.HandleLogout)

		api.GET("/security/:domain", security.SecurityScanHandler)

		api.Use(auth.Middleware())
		{
			api.GET("/register", registerToken)

			api.GET("/profile", getProfile)
			api.GET("/api-token", getApiToken)
		}
	}

	tokenAPI := r.Group("/api")
	{
		tokenAPI.Use(auth.RequireToken())
		{
			// tokenAPI.POST("/sync", persistData)
			// tokenAPI.POST("/dump", dumpData)
		}
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	if err := database.InitDatabase(); err != nil {
		log.Fatal(err)
	}
	db = database.GetDB()

	// Initialize the OIDC configuration
	if err := auth.InitOIDC(); err != nil {
		log.Printf("Failed to initialize OIDC: %v", err)
		// log.Fatalf("Failed to initialize OIDC: %v", err)
	}

	if err := session.Init(); err != nil {
		log.Fatalf("Unable to initalize session storage: %v", err)
	}

	r := gin.Default()

	setupRoutes(r)

	spaDirectory := utils.GetEnv("WEB_DIR", "../web")

	r.Use(spa.Middleware("/", spaDirectory))

	r.Run(":8080")
}
