package main

import (
	"log"

	"github.com/norskhelsenett/chase/auth"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/handlers"
	"github.com/norskhelsenett/chase/security"
	"github.com/norskhelsenett/chase/servers"
	"github.com/norskhelsenett/chase/session"
	"github.com/norskhelsenett/chase/spa"
	"github.com/norskhelsenett/chase/utils"

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
		api.GET("/screenshot/:domain", security.ScreenshotHandler)

		api.Use(auth.Middleware())
		{
			api.POST("/servers", servers.AddServer)
			api.GET("/servers", servers.GetServersWithSecurityStatus)
			api.PUT("/servers/:id", servers.UpdateServer)
			api.GET("/servers/:id", servers.GetServer)
			api.GET("/servers/:id/report", security.LastSecurityScanHandler)
			api.GET("/servers/:id/results", servers.GetServerResults)
			api.POST("/servers/:id/force-check", servers.ForceCheckServer)

			api.POST("/batch/start", security.StartBatchHandler)
			api.GET("/batch/:jobID/status", security.GetBatchStatusHandler)
			api.POST("/batch/:jobID/cancel", security.CancelBatchHandler)
			api.GET("/batch/list", security.ListBatchesHandler)

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
	db.AutoMigrate(&servers.Server{}, &servers.PingResult{})
	db.AutoMigrate(&security.BatchJobStore{}, &security.BatchResultStore{})
	security.InitDatabase()

	go servers.StartMonitoring()

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
