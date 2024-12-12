package session

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/types"
	"gorm.io/gorm"
)

// Session represents the session model in the database
type Session struct {
	gorm.Model
	SessionID string `gorm:"uniqueIndex"`
	Email     string `gorm:"index"`
	Name      string
	ExpiresAt time.Time
}

func Init() error {
	// Perform auto-migration for the Session model
	db := database.GetDB()
	if err := db.AutoMigrate(&Session{}); err != nil {
		return err
	}
	return nil
}

func GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func GetSession(sessionID string) (types.SessionInfo, bool) {
	db := database.GetDB()
	var session Session
	result := db.Where("session_id = ?", sessionID).First(&session)
	if result.Error != nil {
		return types.SessionInfo{}, false
	}

	sessionInfo := types.SessionInfo{
		UserInfo: types.UserInfo{
			Sub:   session.Email,
			Name:  session.Name,
			Email: session.Email,
		},
		Exp: session.ExpiresAt,
	}

	return sessionInfo, true
}

func SetSession(sessionID string, userInfo types.UserInfo, expiration time.Time) error {
	db := database.GetDB()
	session := Session{
		SessionID: sessionID,
		Email:     userInfo.Email,
		Name:      userInfo.Name,
		ExpiresAt: expiration,
	}

	result := db.Create(&session)
	return result.Error
}

func UpdateSession(sessionID string, userInfo types.UserInfo) error {
	db := database.GetDB()
	result := db.Model(&Session{}).Where("session_id = ?", sessionID).Updates(map[string]interface{}{
		"email": userInfo.Email,
		"name":  userInfo.Name,
	})
	return result.Error
}

func DeleteSession(sessionID string) error {
	db := database.GetDB()
	result := db.Where("session_id = ?", sessionID).Delete(&Session{})
	return result.Error
}

func CleanupExpiredSessions() {
	db := database.GetDB()
	db.Where("expires_at < ?", time.Now()).Delete(&Session{})
}

func GetSessionByEmail(email string) (types.SessionInfo, bool) {
	db := database.GetDB()
	var session Session
	result := db.Where("email = ?", email).Order("created_at DESC").First(&session)
	if result.Error != nil {
		return types.SessionInfo{}, false
	}

	sessionInfo := types.SessionInfo{
		UserInfo: types.UserInfo{
			Sub:   session.Email,
			Name:  session.Name,
			Email: session.Email,
		},
		Exp: session.ExpiresAt,
	}

	return sessionInfo, true
}
