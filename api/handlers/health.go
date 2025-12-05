package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
)

var bootTime time.Time

func InitHealth(start time.Time) {
	bootTime = start
}

func LivenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"uptime": time.Since(bootTime).String(),
	})
}

func readinessStatus() error {
	db := database.GetDB()
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func ReadinessProbe(c *gin.Context) {
	if err := readinessStatus(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "degraded",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

func HealthProbe(c *gin.Context) {
	if err := readinessStatus(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "degraded",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"uptime": time.Since(bootTime).String(),
	})
}
