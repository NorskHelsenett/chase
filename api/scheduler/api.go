package scheduler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes adds scheduler API endpoints to the router group.
func (s *Scheduler) RegisterRoutes(rg *gin.RouterGroup) {
	jobs := rg.Group("/jobs")
	{
		jobs.GET("", s.handleListJobs)
		jobs.GET("/stream", s.handleStreamJobs)
		jobs.GET("/:name", s.handleGetJob)
		jobs.POST("/:name/trigger", s.handleTrigger)
		jobs.POST("/:name/cancel", s.handleCancel)
		jobs.GET("/:name/logs", s.handleGetLogs)
	}
	rg.GET("/system-stats", s.handleSystemStats)
}

// handleStreamJobs streams the full jobs snapshot over SSE so the UI sees live
// progress (and elapsed time) without polling. Snapshots are emitted on change
// at ~1s granularity, with a periodic keepalive comment.
func (s *Scheduler) handleStreamJobs(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	ctx := c.Request.Context()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	var last string
	send := func() {
		data, err := json.Marshal(s.ListJobs())
		if err != nil {
			return
		}
		if string(data) == last {
			return
		}
		last = string(data)
		c.Writer.WriteString(fmt.Sprintf("event: jobs\ndata: %s\n\n", data))
		c.Writer.Flush()
	}

	send() // initial snapshot

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			send()
		case <-keepalive.C:
			c.Writer.WriteString(": ping\n\n")
			c.Writer.Flush()
		}
	}
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

	// Database size (Postgres) — pg_database_size returns bytes for the
	// currently connected database. Falls back to 0 on error.
	var dbSize int64
	if err := s.db.Raw(`SELECT pg_database_size(current_database())`).Scan(&dbSize).Error; err != nil {
		dbSize = 0
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
