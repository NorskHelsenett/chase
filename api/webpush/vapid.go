package webpush

import (
	"crypto/elliptic"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"

	webpush "github.com/SherClockHolmes/webpush-go"
	"gorm.io/gorm"
)

// InitDatabase migrates all webpush related tables
func InitDatabase(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&VAPIDKeys{},
		&PushSubscription{},
		&NotificationPreference{},
		&NotificationLog{},
	); err != nil {
		return fmt.Errorf("failed to migrate webpush tables: %v", err)
	}

	// Ensure VAPID keys exist
	if err := ensureVAPIDKeys(db); err != nil {
		return fmt.Errorf("failed to ensure VAPID keys: %v", err)
	}

	return nil
}

// ensureVAPIDKeys checks if VAPID keys exist, and generates them if not
func ensureVAPIDKeys(db *gorm.DB) error {
	var count int64
	if err := db.Model(&VAPIDKeys{}).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		log.Println("No VAPID keys found, generating new keys...")
		publicKey, privateKey, err := generateVAPIDKeys()
		if err != nil {
			return fmt.Errorf("failed to generate VAPID keys: %v", err)
		}

		keys := VAPIDKeys{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		}

		if err := db.Create(&keys).Error; err != nil {
			return fmt.Errorf("failed to save VAPID keys: %v", err)
		}

		log.Printf("Generated and stored new VAPID keys (public key: %s...)", publicKey[:20])
	} else {
		log.Println("VAPID keys already exist")
	}

	return nil
}

// generateVAPIDKeys generates a new VAPID key pair using the webpush-go library
func generateVAPIDKeys() (publicKey, privateKey string, err error) {
	return webpush.GenerateVAPIDKeys()
}

// GetVAPIDKeys retrieves the current VAPID keys from the database
func GetVAPIDKeys(db *gorm.DB) (*VAPIDKeys, error) {
	var keys VAPIDKeys
	if err := db.First(&keys).Error; err != nil {
		return nil, err
	}
	return &keys, nil
}

// GetPublicVAPIDKey returns just the public key (for client-side use)
func GetPublicVAPIDKey(db *gorm.DB) (string, error) {
	keys, err := GetVAPIDKeys(db)
	if err != nil {
		return "", err
	}
	return keys.PublicKey, nil
}

// RegenerateVAPIDKeys generates and stores new VAPID keys
// WARNING: This will invalidate all existing subscriptions!
func RegenerateVAPIDKeys(db *gorm.DB) error {
	publicKey, privateKey, err := generateVAPIDKeys()
	if err != nil {
		return fmt.Errorf("failed to generate new VAPID keys: %v", err)
	}

	// Delete old keys
	if err := db.Where("1 = 1").Delete(&VAPIDKeys{}).Error; err != nil {
		return fmt.Errorf("failed to delete old keys: %v", err)
	}

	// Create new keys
	keys := VAPIDKeys{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}

	if err := db.Create(&keys).Error; err != nil {
		return fmt.Errorf("failed to save new VAPID keys: %v", err)
	}

	// Delete all existing subscriptions since they're now invalid
	if err := db.Where("1 = 1").Delete(&PushSubscription{}).Error; err != nil {
		log.Printf("Warning: failed to delete old subscriptions: %v", err)
	}

	log.Println("Successfully regenerated VAPID keys and cleared old subscriptions")
	return nil
}

// Helper function to convert base64 URL-safe to standard base64 if needed
func decodeBase64URLOrStandard(s string) ([]byte, error) {
	// Try URL-safe first
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err == nil {
		return data, nil
	}

	// Try standard base64
	return base64.StdEncoding.DecodeString(s)
}

// ValidatePrivateKey checks if the private key is valid
func ValidatePrivateKey(privateKeyB64 string) error {
	privateKeyBytes, err := decodeBase64URLOrStandard(privateKeyB64)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %v", err)
	}

	if len(privateKeyBytes) != 32 {
		return fmt.Errorf("private key must be 32 bytes, got %d", len(privateKeyBytes))
	}

	// Check if it's a valid scalar for P-256
	curve := elliptic.P256()
	d := new(big.Int).SetBytes(privateKeyBytes)
	if d.Cmp(curve.Params().N) >= 0 {
		return fmt.Errorf("private key is out of range")
	}

	return nil
}
