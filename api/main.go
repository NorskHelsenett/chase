package main

import (
	"fmt"
	"log"
	"os"
	"strings"

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

func loadEnv() error {
	// Try to load from .env file first
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, checking ENV_FILE variable")

		// Check if we have env content in an environment variable
		if envContent := os.Getenv("ENV_FILE"); envContent != "" {
			// Create a reader from the string content
			reader := strings.NewReader(envContent)

			// Parse returns map[string]string and error
			envMap, err := godotenv.Parse(reader)
			if err != nil {
				return fmt.Errorf("failed to parse ENV_FILE content: %w", err)
			}

			// Set the environment variables
			for key, value := range envMap {
				os.Setenv(key, value)
			}

			log.Println("Loaded environment from ENV_FILE variable")
			return nil
		}

		log.Println("No ENV_FILE content found, using system environment variables")
	}

	return nil
}

func main() {
	if err := loadEnv(); err != nil {
		log.Fatal(err)
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
