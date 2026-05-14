package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/norskhelsenett/chase/auth"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/handlers"
	"github.com/norskhelsenett/chase/scheduler"
	"github.com/norskhelsenett/chase/security"
	"github.com/norskhelsenett/chase/servers"
	"github.com/norskhelsenett/chase/session"
	"github.com/norskhelsenett/chase/spa"
	"github.com/norskhelsenett/chase/utils"
	"github.com/norskhelsenett/chase/webpush"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var db *gorm.DB
var appStart = time.Now()
var sched *scheduler.Scheduler

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
			api.GET("/servers/geo", servers.GetServersGeo)
			api.GET("/servers/:id/favicon", servers.GetServerFavicon)
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

			// Scheduler routes
			sched.RegisterRoutes(api)
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

	// Schema migrations run synchronously — must complete before serving API routes
	servers.AutoMigrate(db)
	db.AutoMigrate(&security.BatchJobStore{}, &security.BatchResultStore{})
	security.InitDatabase()
	security.SetMaxParallelScreenshots(2)

	if err := webpush.InitDatabase(db); err != nil {
		log.Printf("Failed to initialize web push: %v", err)
	} else {
		log.Println("Web push notification system initialized")
	}

	if err := auth.InitOIDC(); err != nil {
		log.Printf("Failed to initialize OIDC: %v", err)
	}

	if err := session.Init(); err != nil {
		log.Fatalf("Unable to initalize session storage: %v", err)
	}

	// One-shot atomic import from a legacy SQLite chase.db, gated by
	// MIGRATE_FROM_SQLITE. No-op when unset, missing, or already applied.
	if err := migrateFromSQLite(db); err != nil {
		log.Fatalf("SQLite import failed: %v", err)
	}

	// Initialize scheduler and register all jobs (before routes so handlers can reference it)
	sched = scheduler.New(db)
	registerJobs(sched)
	sched.Start()

	// Build initial geo cache before serving
	go servers.RebuildGeoResponseCache()

	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(securityHeaders())

	r.GET("/livez", handlers.LivenessProbe)
	r.GET("/readyz", handlers.ReadinessProbe)
	r.GET("/healthz", handlers.HealthProbe)

	setupRoutes(r)

	spaDirectory := utils.GetEnv("WEB_DIR", "../web")
	r.Use(spa.Middleware("/", spaDirectory))

	log.Println("Application routes registered")

	r.Run(":8080")
}

func getMonitoringInterval() time.Duration {
	if s, err := strconv.Atoi(os.Getenv("MONITORING_INTERVAL")); err == nil && s > 0 {
		return time.Duration(s) * time.Minute
	}
	return 1 * time.Minute
}

func registerJobs(s *scheduler.Scheduler) {
	// Server monitoring — ping all active servers due for check
	s.Register("server-monitoring", "Ping all active servers due for check",
		scheduler.Schedule{Interval: getMonitoringInterval()},
		func(ctx context.Context, progress func(string)) (string, error) {
			servers.RunMonitoring()
			return "monitoring cycle complete", nil
		},
	)

	// Geo cache entry refresh — refresh stale geo IP entries
	s.Register("geo-cache-refresh", "Refresh stale geo IP cache entries",
		scheduler.Schedule{TimeOfDay: &scheduler.TimeOfDay{Hour: 3, Minute: 0}},
		func(ctx context.Context, progress func(string)) (string, error) {
			servers.RefreshStaleGeoEntries()
			return "geo entries refreshed", nil
		},
	)

	// Geo response cache rebuild — rebuild full geo response cache
	s.Register("geo-response-rebuild", "Rebuild server geo response cache",
		scheduler.Schedule{Interval: 1 * time.Hour},
		func(ctx context.Context, progress func(string)) (string, error) {
			servers.RebuildGeoResponseCache()
			return "geo response cache rebuilt", nil
		},
	)

	// Database cleanup — dedup, prune old data, vacuum
	s.Register("database-cleanup", "Dedup screenshots/reports, prune old data, vacuum",
		scheduler.Schedule{TimeOfDay: &scheduler.TimeOfDay{Hour: 2, Minute: 0}},
		func(ctx context.Context, progress func(string)) (string, error) {
			return security.RunDatabaseCleanup(), nil
		},
	)

	// Aggregate & prune pings — three-tier retention policy
	s.Register("aggregate-prune-pings", "Three-tier ping retention: raw -> hourly -> daily",
		scheduler.Schedule{TimeOfDay: &scheduler.TimeOfDay{Hour: 2, Minute: 30}},
		func(ctx context.Context, progress func(string)) (string, error) {
			servers.AggregateAndPrunePings()
			return "ping aggregation complete", nil
		},
	)

	// Backfill thumbnails — generate missing screenshot thumbnails
	s.Register("backfill-thumbnails", "Generate thumbnails for screenshots missing them",
		scheduler.Schedule{TimeOfDay: &scheduler.TimeOfDay{Hour: 4, Minute: 0}},
		func(ctx context.Context, progress func(string)) (string, error) {
			return security.BackfillThumbnails(), nil
		},
	)

	// Batch security scan — scan all eligible servers (manual only)
	s.Register("batch-security-scan", "Security scan + screenshot for all active servers",
		scheduler.Schedule{Manual: true},
		func(ctx context.Context, progress func(string)) (string, error) {
			return security.RunBatchSecurityScan(ctx, progress)
		},
	)

	// Inactive server recheck — ping deactivated servers to see if they're back
	s.Register("inactive-server-recheck", "Ping deactivated servers to check if they're back online",
		scheduler.Schedule{TimeOfDay: &scheduler.TimeOfDay{Hour: 5, Minute: 0}},
		func(ctx context.Context, progress func(string)) (string, error) {
			return servers.RecheckInactiveServers(ctx, progress)
		},
	)
}
