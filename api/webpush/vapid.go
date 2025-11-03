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
	// NOTE: The webpush-go library returns keys in REVERSE order:
	// First return value is actually the PRIVATE key (32 bytes)
	// Second return value is actually the PUBLIC key (65 bytes)
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return "", "", err
	}
	// Return them in the correct order: public, private
	return pub, priv, nil
}

// GetVAPIDKeys retrieves the current VAPID keys from the database
func GetVAPIDKeys(db *gorm.DB) (*VAPIDKeys, error) {
	var keys VAPIDKeys
	if err := db.First(&keys).Error; err != nil {
		return nil, err
	}

	// Validate and normalize the keys
	publicKey, privateKey, err := normalizeVAPIDKeyPair(keys.PublicKey, keys.PrivateKey)
	if err != nil {
		log.Printf("Warning: Invalid VAPID keys in database: %v. Regenerating...", err)
		// Keys are invalid, regenerate them
		if regErr := RegenerateVAPIDKeys(db); regErr != nil {
			return nil, fmt.Errorf("failed to regenerate invalid VAPID keys: %v", regErr)
		}
		// Retrieve the new keys
		if err := db.First(&keys).Error; err != nil {
			return nil, err
		}
		return &keys, nil
	}

	// Update keys in database if they were normalized
	if publicKey != keys.PublicKey || privateKey != keys.PrivateKey {
		keys.PublicKey = publicKey
		keys.PrivateKey = privateKey
		if err := db.Save(&keys).Error; err != nil {
			log.Printf("Warning: Failed to save normalized keys: %v", err)
		}
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

// normalizeVAPIDKeyPair validates and normalizes VAPID keys to the format expected by webpush-go
func normalizeVAPIDKeyPair(publicKey, privateKey string) (string, string, error) {
	// Decode and validate private key (must be 32 bytes)
	privBytes, err := decodeBase64URLOrStandard(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("invalid private key encoding: %v", err)
	}

	if len(privBytes) != 32 {
		return "", "", fmt.Errorf("private key must be 32 bytes, got %d", len(privBytes))
	}

	// Decode and validate public key (must be 65 bytes uncompressed)
	pubBytes, err := decodeBase64URLOrStandard(publicKey)
	if err != nil {
		return "", "", fmt.Errorf("invalid public key encoding: %v", err)
	}

	if len(pubBytes) != 65 {
		return "", "", fmt.Errorf("public key must be 65 bytes (uncompressed P-256), got %d", len(pubBytes))
	}

	if pubBytes[0] != 0x04 {
		return "", "", fmt.Errorf("public key must start with 0x04 (uncompressed format), got 0x%02x", pubBytes[0])
	}

	// Validate that it's a valid P-256 point
	curve := elliptic.P256()
	x := new(big.Int).SetBytes(pubBytes[1:33])
	y := new(big.Int).SetBytes(pubBytes[33:65])

	if !curve.IsOnCurve(x, y) {
		return "", "", fmt.Errorf("public key is not a valid P-256 curve point")
	}

	// Validate private key is in valid range
	d := new(big.Int).SetBytes(privBytes)
	if d.Cmp(curve.Params().N) >= 0 {
		return "", "", fmt.Errorf("private key is out of range")
	}

	// Normalize to base64 URL-safe encoding without padding
	normalizedPub := base64.RawURLEncoding.EncodeToString(pubBytes)
	normalizedPriv := base64.RawURLEncoding.EncodeToString(privBytes)

	return normalizedPub, normalizedPriv, nil
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
