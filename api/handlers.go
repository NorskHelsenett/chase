package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/handlers"
	"github.com/norskhelsenett/chase/types"
	"github.com/norskhelsenett/chase/utils"
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
		return
	}
	admin := types.IsAdmin(database.GetDB(), user.Email)
	c.JSON(http.StatusOK, gin.H{
		"name":     user.Name,
		"email":    user.Email,
		"is_admin": admin,
	})
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

// func getUserFromToken(c *gin.Context) (*types.User, error) {
// 	apiToken := c.GetHeader("x-api-token")
// 	if apiToken == "" {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing API token"})
// 		return nil, fmt.Errorf("missing API token")
// 	}

// 	var user types.User
// 	if err := db.Where("api_token = ?", apiToken).First(&user).Error; err != nil {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API token"})
// 		return nil, fmt.Errorf("invalid API token")
// 	}

// 	return &user, nil
// }
