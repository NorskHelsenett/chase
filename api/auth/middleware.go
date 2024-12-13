package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/session"
	"github.com/norskhelsenett/chase/types"
)

func RequireToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiToken := c.GetHeader("x-api-token")
		if apiToken != "" {
			exists, err := isTokenInDatabase(apiToken)
			if err == nil || exists {
				c.Next()
				return
			}
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
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
	db := database.GetDB()
	var exists bool
	err := db.Model(&types.User{}).
		Select("count(*) > 0").
		Where("api_token = ?", token).
		Scan(&exists).
		Error
	return exists, err
}
