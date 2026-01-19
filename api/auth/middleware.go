package auth

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/session"
	"github.com/norskhelsenett/chase/types"
)

func RequireToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiToken := c.GetHeader("x-api-token")
		if apiToken == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Basic format validation before DB lookup
		if len(apiToken) != 35 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		exists, err := isTokenInDatabase(apiToken)
		if err == nil && exists {
			c.Next()
			return
		}

		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Check API token first
		if apiToken := c.GetHeader("x-api-token"); apiToken != "" {
			// Basic format validation before DB lookup
			if len(apiToken) != 35 {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			exists, err := isTokenInDatabase(apiToken)
			if err == nil && exists {
				c.Next()
				return
			}
		}

		// Fall back to session cookie
		sessionID, err := c.Cookie("session")
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		sessionInfo, ok := session.GetSession(sessionID)
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Check expiration
		if time.Now().After(sessionInfo.Exp) {
			c.SetCookie("session", "", -1, "/", "", true, true)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Set user context
		c.Set("email", sessionInfo.UserInfo.Email)
		c.Next()
	}
}

func isTokenInDatabase(token string) (bool, error) {
	// Check against environment variable first (faster than DB lookup)
	if key := os.Getenv("CHASE_SECRET_KEY"); key != "" && token == key {
		return true, nil
	}

	// Only query database if token isn't the secret key
	db := database.GetDB()
	var exists bool
	err := db.Model(&types.User{}).
		Select("1").
		Where("api_token = ?", token).
		Limit(1).
		Scan(&exists).
		Error

	return exists, err
}
