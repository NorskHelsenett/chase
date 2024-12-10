package main

import (
	"fmt"
	"net/http"

	"git.torden.tech/jonasbg/fit/database"
	"git.torden.tech/jonasbg/fit/handlers"
	"git.torden.tech/jonasbg/fit/types"
	"git.torden.tech/jonasbg/fit/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getApiToken(c *gin.Context) {
	user, err := getUserFromCookie(c)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
	}
	db = database.GetDB()
	var apiToken string
	if err := db.Model(&user).Select("api_token").Scan(&apiToken).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "API token not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API token"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_token": apiToken})
}

func getProfile(c *gin.Context) {
	user, err := getUserFromCookie(c)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
	}
	c.JSON(http.StatusOK, user)
}

func registerToken(c *gin.Context) {
	apiToken := utils.GenerateAPIToken()

	user, err := getUserFromCookie(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	if err := db.Model(&user).Update("api_token", apiToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update API token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"x-api-token": apiToken})
}

func getUserFromCookie(c *gin.Context) (*types.User, error) {
	email, err := handlers.GetEmail(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing API token"})
		return nil, fmt.Errorf("unauthorized")
	}

	var user types.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API token"})
		return nil, fmt.Errorf("invalid API token")
	}

	return &user, nil
}

func getUserFromToken(c *gin.Context) (*types.User, error) {
	apiToken := c.GetHeader("x-api-token")
	if apiToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing API token"})
		return nil, fmt.Errorf("missing API token")
	}

	var user types.User
	if err := db.Where("api_token = ?", apiToken).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API token"})
		return nil, fmt.Errorf("invalid API token")
	}

	return &user, nil
}
