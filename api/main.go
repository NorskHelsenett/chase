package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/norskhelsenett/chase/auth"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/handlers"
	"github.com/norskhelsenett/chase/security"
	"github.com/norskhelsenett/chase/servers"
	"github.com/norskhelsenett/chase/session"
	"github.com/norskhelsenett/chase/spa"
	"github.com/norskhelsenett/chase/utils"
	"github.com/norskhelsenett/chase/webpush"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var db *gorm.DB
var appStart = time.Now()

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
		api.GET("/security/:domain/stream", security.SecurityScanSSEHandler)
		api.GET("/screenshot/:domain", security.ScreenshotHandler)
		api.POST("/webhooks/servers", auth.RequireToken(), servers.AddServerFromWebhook)

		api.Use(auth.Middleware())
		{
			api.POST("/servers", servers.AddServer)
			api.GET("/servers", servers.GetServersWithSecurityStatus)
			api.GET("/servers/pings/stream", servers.PingStreamSSE)
			api.PUT("/servers/:id", servers.UpdateServer)
			api.PATCH("/servers/:id", servers.PatchServer)
			api.GET("/servers/:id", servers.GetServer)
			api.DELETE("/servers/:id", servers.DeleteServer)
			api.GET("/servers/:id/report", security.LastSecurityScanHandler)
			api.GET("/servers/:id/pings", servers.GetServerResults)
			api.POST("/servers/:id/force-check", servers.ForceCheckServer)
			api.POST("/servers/batch-import", servers.BatchImportServers)

			api.POST("/batch/start", security.StartBatchHandler)
			api.GET("/batch/:jobID/status", security.GetBatchStatusHandler)
			api.POST("/batch/:jobID/cancel", security.CancelBatchHandler)
			api.GET("/batch/list", security.ListBatchesHandler)

			api.GET("/register", registerToken)

			api.GET("/profile", getProfile)
			api.GET("/api-token", getApiToken)

			// Web Push notification routes
			pushHandler := webpush.NewHandler(db)
			pushHandler.RegisterRoutes(api)
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
	handlers.InitHealth(appStart)
	servers.AutoMigrate(db)
	db.AutoMigrate(&security.BatchJobStore{}, &security.BatchResultStore{})
	security.InitDatabase()
	security.SetMaxParallelScreenshots(2)

	// Initialize web push notification system
	if err := webpush.InitDatabase(db); err != nil {
		log.Printf("Failed to initialize web push: %v", err)
	} else {
		log.Println("Web push notification system initialized")
	}

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

	r.GET("/livez", handlers.LivenessProbe)
	r.GET("/readyz", handlers.ReadinessProbe)
	r.GET("/healthz", handlers.HealthProbe)

	setupRoutes(r)

	spaDirectory := utils.GetEnv("WEB_DIR", "../web")

	r.Use(spa.Middleware("/", spaDirectory))

	r.Run(":8080")
}
