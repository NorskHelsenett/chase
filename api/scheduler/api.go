package scheduler

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes adds scheduler API endpoints to the router group.
func (s *Scheduler) RegisterRoutes(rg *gin.RouterGroup) {
	jobs := rg.Group("/jobs")
	{
		jobs.GET("", s.handleListJobs)
		jobs.GET("/:name", s.handleGetJob)
		jobs.POST("/:name/trigger", s.handleTrigger)
		jobs.POST("/:name/cancel", s.handleCancel)
		jobs.GET("/:name/logs", s.handleGetLogs)
	}
	rg.GET("/system-stats", s.handleSystemStats)
}

func (s *Scheduler) handleListJobs(c *gin.Context) {
	c.JSON(http.StatusOK, s.ListJobs())
}

func (s *Scheduler) handleGetJob(c *gin.Context) {
	name := c.Param("name")
	info, ok := s.GetJob(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (s *Scheduler) handleTrigger(c *gin.Context) {
	name := c.Param("name")
	if err := s.Trigger(name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "triggered", "job": name})
}

func (s *Scheduler) handleCancel(c *gin.Context) {
	name := c.Param("name")
	if err := s.Cancel(name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled", "job": name})
}

func (s *Scheduler) handleSystemStats(c *gin.Context) {
	var activeServers, inactiveServers, totalPings, userCount int64

	s.db.Table("servers").Where("active = ? AND deleted_at IS NULL", true).Count(&activeServers)
	s.db.Table("servers").Where("active = ? AND deleted_at IS NULL", false).Count(&inactiveServers)
	s.db.Table("ping_results").Count(&totalPings)
	s.db.Table("users").Count(&userCount)

	// Database file size
	var dbSize int64
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "chase.db"
	}
	if info, err := os.Stat(dbPath); err == nil {
		dbSize = info.Size()
	}

	// Running jobs count
	jobs := s.ListJobs()
	running := 0
	for _, j := range jobs {
		if j.Status == StatusRunning {
			running++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"active_servers":   activeServers,
		"inactive_servers": inactiveServers,
		"total_pings":      totalPings,
		"users":            userCount,
		"database_bytes":   dbSize,
		"total_jobs":       len(jobs),
		"running_jobs":     running,
	})
}

func (s *Scheduler) handleGetLogs(c *gin.Context) {
	name := c.Param("name")
	limit := 20
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	var records []JobRunRecord
	s.db.Where("job_name = ?", name).
		Order("started_at DESC").
		Limit(limit).
		Find(&records)

	c.JSON(http.StatusOK, records)
}
