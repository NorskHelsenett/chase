package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/norskhelsenett/chase/auth"
	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/session"
	"github.com/norskhelsenett/chase/types"
	"github.com/norskhelsenett/chase/utils"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func HandleLogin(c *gin.Context) {
	url := auth.Config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusFound, url)
}

func HandleCallback(c *gin.Context) {
	code := c.Query("code")
	token, err := auth.Config.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	userInfo, err := auth.GetUserInfo(token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	sessionID, err := session.GenerateSessionID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session ID"})
		return
	}

	// Set expiration time to 72 hours from now
	expirationTime := time.Now().Add(170 * time.Hour)

	// Set session with expiration time
	err = session.SetSession(sessionID, *userInfo, expirationTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set session"})
		return
	}

	var db = database.GetDB()
	var user types.User
	isNewUser := false

	// Attempt to find the user by email
	result := db.Where(types.User{Email: userInfo.Email}).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// User not found, create a new one
			isNewUser = true
			user = types.User{
				Name:     userInfo.Name,
				Email:    userInfo.Email,
				APIToken: utils.GenerateAPIToken(),
			}

			if err := db.Create(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query user"})
			return
		}
	} else {
		// User found, update their information
		user.Name = userInfo.Name
		if user.APIToken == "" {
			// Only generate a new API token if one doesn't exist
			user.APIToken = utils.GenerateAPIToken()
		}
		if err := db.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}
	}

	// Set cookie with the same expiration time as the session
	maxAge := int(time.Until(expirationTime).Seconds())
	c.SetCookie("session", sessionID, maxAge, "/", "", c.Request.TLS != nil, true)

	if isNewUser {
		// Redirect new users to a welcome page, passing the API token
		c.Redirect(http.StatusFound, fmt.Sprintf("/welcome?api_token=%s", user.APIToken))
	} else {
		c.Redirect(http.StatusFound, "/")
	}
}

func HandleProtected(c *gin.Context) {
	sessionInfo, _ := c.Get("session")
	sessionData, ok := sessionInfo.(types.SessionInfo)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email": sessionData.UserInfo.Email,
		"name":  sessionData.UserInfo.Name,
	})
}

func HandleLogout(c *gin.Context) {
	c.SetCookie("session", "", -1, "/", "", c.Request.TLS != nil, true)
	c.Redirect(http.StatusFound, "/")
}

func GetEmail(c *gin.Context) (string, error) {
	// Check if email is already in the context
	if email, exists := c.Get("email"); exists {
		if emailStr, ok := email.(string); ok {
			return emailStr, nil
		}
		return "", errors.New("email in context is not a string")
	}

	// If not in context, check the session
	sessionID, err := c.Cookie("session")
	if err != nil {
		return "", errors.New("session cookie not found")
	}

	sessionInfo, ok := session.GetSession(sessionID)
	if !ok || time.Now().After(sessionInfo.Exp) {
		return "", errors.New("invalid or expired session")
	}

	// Store session info and email in context for future use
	c.Set("session", sessionInfo)
	c.Set("email", sessionInfo.UserInfo.Email)

	return sessionInfo.UserInfo.Email, nil
}
