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

func loadEnv() error {
	// Try to load from .env file
	if err := godotenv.Load(); err == nil {
		log.Println("Loaded environment from .env file")
	} else {
		log.Println("No .env file found, checking ENV_FILE variable")
	}

	// Check if we have env file path in environment variable
	if envPath := os.Getenv("ENV_FILE"); envPath != "" {
		// Read the file content
		content, err := os.ReadFile(envPath)
		if err != nil {
			return fmt.Errorf("failed to read file at %s: %w", envPath, err)
		}

		reader := strings.NewReader(string(content))
		envMap, err := godotenv.Parse(reader)
		if err != nil {
			return fmt.Errorf("failed to parse env file at %s: %w", envPath, err)
		}

		// Set the environment variables
		for key, value := range envMap {
			os.Setenv(key, value)
		}
		log.Println("Loaded environment from", envPath)
	}

	return nil
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent browsers from performing MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Protect against clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Enable browser's XSS filter
		c.Header("X-XSS-Protection", "1; mode=block")

		// Control how much information the browser includes with referrers
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Enforce HTTPS
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Permissions Policy (formerly Feature-Policy)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

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
	security.SetMaxParallelScreenshots(2)

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
	r.Use(gin.Recovery())
	r.Use(securityHeaders())

	setupRoutes(r)

	spaDirectory := utils.GetEnv("WEB_DIR", "../web")

	r.Use(spa.Middleware("/", spaDirectory))

	r.Run(":8080")
}
