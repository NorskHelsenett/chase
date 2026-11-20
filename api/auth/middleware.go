package auth

import (
	"crypto/subtle"
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

		id, err := lookupToken(apiToken)
		if err == nil && id.ok {
			if id.master {
				c.Set(ctxMasterKey, true)
			} else {
				c.Set("email", id.email)
			}
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

			id, err := lookupToken(apiToken)
			if err == nil && id.ok {
				// Establish identity the same way the session path does, so
				// email-dependent handlers work for token-authenticated calls.
				if id.master {
					c.Set(ctxMasterKey, true)
				} else {
					c.Set("email", id.email)
				}
				c.Next()
				return
			}
			c.AbortWithStatus(http.StatusUnauthorized)
			return
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

// ctxMasterKey marks a request authenticated with CHASE_SECRET_KEY.
const ctxMasterKey = "auth.master"

type tokenIdentity struct {
	ok     bool
	master bool
	email  string
}

// lookupToken resolves an API token to the identity it belongs to.
func lookupToken(token string) (tokenIdentity, error) {
	// Constant-time compare against the master key to avoid a timing oracle.
	if key := os.Getenv("CHASE_SECRET_KEY"); key != "" &&
		subtle.ConstantTimeCompare([]byte(token), []byte(key)) == 1 {
		return tokenIdentity{ok: true, master: true}, nil
	}

	var user types.User
	err := database.GetDB().
		Select("email").
		Where("api_token = ?", token).
		Limit(1).
		Find(&user).
		Error
	if err != nil {
		return tokenIdentity{}, err
	}
	if user.Email == "" {
		return tokenIdentity{}, nil
	}
	return tokenIdentity{ok: true, email: user.Email}, nil
}

// RequireAdmin must run after Middleware. It allows the master key and the
// admin user (types.IsAdmin), and rejects everyone else with 403.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool(ctxMasterKey) {
			c.Next()
			return
		}
		email := c.GetString("email")
		if email == "" || !types.IsAdmin(database.GetDB(), email) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin privileges required"})
			return
		}
		c.Next()
	}
}
