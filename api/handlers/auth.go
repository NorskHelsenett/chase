package handlers

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
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

const (
	stateCookie    = "oauth_state"
	verifierCookie = "oauth_verifier"
	// The login/callback round-trip should complete well within this window.
	authFlowMaxAge = 600
)

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func HandleLogin(c *gin.Context) {
	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
		return
	}
	verifier := oauth2.GenerateVerifier()

	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(stateCookie, state, authFlowMaxAge, "/", "", secure, true)
	c.SetCookie(verifierCookie, verifier, authFlowMaxAge, "/", "", secure, true)

	url := auth.Config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	c.Redirect(http.StatusFound, url)
}

func HandleCallback(c *gin.Context) {
	secure := c.Request.TLS != nil
	clearAuthFlowCookies := func() {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(stateCookie, "", -1, "/", "", secure, true)
		c.SetCookie(verifierCookie, "", -1, "/", "", secure, true)
	}

	expectedState, err := c.Cookie(stateCookie)
	if err != nil || expectedState == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or expired login state, please try logging in again"})
		return
	}
	if subtle.ConstantTimeCompare([]byte(c.Query("state")), []byte(expectedState)) != 1 {
		clearAuthFlowCookies()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	verifier, err := c.Cookie(verifierCookie)
	if err != nil || verifier == "" {
		clearAuthFlowCookies()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or expired login state, please try logging in again"})
		return
	}
	clearAuthFlowCookies()

	code := c.Query("code")
	token, err := auth.Config.Exchange(context.Background(), code, oauth2.VerifierOption(verifier))
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
