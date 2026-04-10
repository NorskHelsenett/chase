package jobs

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/types"
)

// RequireAdmin middleware checks that the requesting user is the first registered user.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		email, exists := c.Get("email")
		if !exists {
			c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
			return
		}

		if !types.IsAdmin(database.GetDB(), email.(string)) {
			c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

// ListJobsHandler returns all registered job definitions.
func ListJobsHandler(c *gin.Context) {
	c.JSON(200, ListJobs())
}

// RunJobHandler manually triggers a job by name.
func RunJobHandler(c *gin.Context) {
	name := c.Param("name")
	logID, err := RunManually(name)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"job_log_id": logID, "status": "started"})
}

// CancelJobHandler cancels a running job.
func CancelJobHandler(c *gin.Context) {
	name := c.Param("name")
	if err := CancelJob(name); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "cancelled"})
}

// GetJobLogsHandler returns logs for jobs with pagination.
func GetJobLogsHandler(c *gin.Context) {
	jobName := c.Query("job")
	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	logs, total, err := GetLogs(jobName, limit, offset)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch job logs"})
		return
	}

	c.JSON(200, gin.H{
		"logs":  logs,
		"total": total,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// RegisterRoutes sets up job-related API routes on the given group.
// All routes require admin access.
func RegisterRoutes(group *gin.RouterGroup) {
	jobRoutes := group.Group("/jobs")
	jobRoutes.Use(RequireAdmin())
	{
		jobRoutes.GET("", ListJobsHandler)
		jobRoutes.POST("/:name/run", RunJobHandler)
		jobRoutes.POST("/:name/cancel", CancelJobHandler)
		jobRoutes.GET("/logs", GetJobLogsHandler)
	}
}
