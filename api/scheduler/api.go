package scheduler

import (
	"net/http"
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
