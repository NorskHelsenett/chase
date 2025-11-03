package webpush

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/hkdf"
)

// Notification represents a web push notification payload
type Notification struct {
	Title   string                 `json:"title"`
	Body    string                 `json:"body"`
	Icon    string                 `json:"icon,omitempty"`
	Badge   string                 `json:"badge,omitempty"`
	Image   string                 `json:"image,omitempty"`
	URL     string                 `json:"url,omitempty"`
	Tag     string                 `json:"tag,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Actions []NotificationAction   `json:"actions,omitempty"`
}

// NotificationAction represents an action button on the notification
type NotificationAction struct {
	Action string `json:"action"`
	Title  string `json:"title"`
	Icon   string `json:"icon,omitempty"`
}

// SendOptions contains options for sending a push notification
type SendOptions struct {
	TTL             int    // Time to live in seconds (default: 2419200 = 4 weeks)
	Urgency         string // low, normal, high, very-low
	Topic           string // For replacing notifications
	VAPIDPublicKey  string
	VAPIDPrivateKey string
}

// SendNotification sends a push notification to a specific subscription
func SendNotification(subscription *PushSubscription, notification *Notification, options *SendOptions) error {
	// Marshal notification to JSON
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %v", err)
	}

	// Encrypt payload
	encryptedPayload, _, _, err := encryptPayload(payload, subscription.P256dh, subscription.Auth)
	if err != nil {
		return fmt.Errorf("failed to encrypt payload: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", subscription.Endpoint, bytes.NewReader(encryptedPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Length", strconv.Itoa(len(encryptedPayload)))

	// Set TTL
	ttl := 2419200 // 4 weeks default
	if options != nil && options.TTL > 0 {
		ttl = options.TTL
	}
	req.Header.Set("TTL", strconv.Itoa(ttl))

	// Set urgency
	urgency := "normal"
	if options != nil && options.Urgency != "" {
		urgency = options.Urgency
	}
	req.Header.Set("Urgency", urgency)

	// Set topic if provided
	if options != nil && options.Topic != "" {
		req.Header.Set("Topic", options.Topic)
	}

	// Generate VAPID authentication header
	if options != nil && options.VAPIDPublicKey != "" && options.VAPIDPrivateKey != "" {
		authHeader, err := generateVAPIDAuthHeader(
			subscription.Endpoint,
			options.VAPIDPublicKey,
			options.VAPIDPrivateKey,
		)
		if err != nil {
			return fmt.Errorf("failed to generate VAPID auth: %v", err)
		}
		req.Header.Set("Authorization", authHeader)
	}

	// Note: With aes128gcm, the salt and server public key are embedded in the payload
	// We do NOT send Crypto-Key or Encryption headers (those are only for legacy aesgcm)

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body for error details
	body, _ := io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode == 201 || resp.StatusCode == 200 {
		return nil // Success
	}

	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		return fmt.Errorf("subscription expired (status %d)", resp.StatusCode)
	}

	return fmt.Errorf("push service error: status %d, body: %s", resp.StatusCode, string(body))
}

// encryptPayload encrypts the notification payload using aes128gcm
func encryptPayload(payload []byte, userPublicKeyB64, userAuthB64 string) ([]byte, []byte, []byte, error) {
	// Decode user public key and auth secret
	userPublicKey, err := base64.RawURLEncoding.DecodeString(userPublicKeyB64)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid user public key: %v", err)
	}

	userAuth, err := base64.RawURLEncoding.DecodeString(userAuthB64)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid auth secret: %v", err)
	}

	// Generate server key pair
	curve := elliptic.P256()
	serverPrivateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate server key: %v", err)
	}

	// Extract server public key (uncompressed format)
	serverPublicKeyBytes := elliptic.Marshal(curve, serverPrivateKey.PublicKey.X, serverPrivateKey.PublicKey.Y)

	// Extract user public key coordinates
	x, y := elliptic.Unmarshal(curve, userPublicKey)
	if x == nil {
		return nil, nil, nil, fmt.Errorf("invalid user public key format")
	}

	// Compute shared secret (ECDH)
	sharedX, _ := curve.ScalarMult(x, y, serverPrivateKey.D.Bytes())
	sharedSecret := sharedX.Bytes()
	// Pad to 32 bytes
	if len(sharedSecret) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(sharedSecret):], sharedSecret)
		sharedSecret = padded
	}

	// Generate random salt (16 bytes)
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate salt: %v", err)
	}

	// Derive encryption key and nonce using HKDF
	prk := hkdf.Extract(sha256.New, sharedSecret, userAuth)

	// Derive key info
	keyInfo := buildInfo("aesgcm", userPublicKey, serverPublicKeyBytes)
	keyReader := hkdf.Expand(sha256.New, prk, keyInfo)
	key := make([]byte, 16)
	if _, err := io.ReadFull(keyReader, key); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to derive key: %v", err)
	}

	// Derive nonce info
	nonceInfo := buildInfo("nonce", userPublicKey, serverPublicKeyBytes)
	nonceReader := hkdf.Expand(sha256.New, prk, nonceInfo)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(nonceReader, nonce); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to derive nonce: %v", err)
	}

	// Pad payload (add padding delimiter and padding)
	paddedPayload := append(payload, 2) // Add padding delimiter
	// Add at least 1 byte of padding
	paddedPayload = append(paddedPayload, 0)

	// Encrypt using AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	ciphertext := aesgcm.Seal(nil, nonce, paddedPayload, nil)

	return ciphertext, salt, serverPublicKeyBytes, nil
}

// buildInfo builds the info parameter for HKDF
func buildInfo(infoType string, userPublicKey, serverPublicKey []byte) []byte {
	info := []byte("Content-Encoding: " + infoType + "\x00")
	return info
}

// generateVAPIDAuthHeader generates the Authorization header for VAPID
func generateVAPIDAuthHeader(endpoint, publicKeyB64, privateKeyB64 string) (string, error) {
	// Parse endpoint to get audience (origin)
	audience := endpoint
	if idx := strings.Index(endpoint[8:], "/"); idx != -1 {
		audience = endpoint[:8+idx]
	}

	// Create JWT header
	header := map[string]string{
		"typ": "JWT",
		"alg": "ES256",
	}

	// Create JWT claims
	claims := map[string]interface{}{
		"aud": audience,
		"exp": time.Now().Add(12 * time.Hour).Unix(),
		"sub": "mailto:noreply@example.com", // You should customize this
	}

	// Encode header and claims
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	unsignedToken := headerB64 + "." + claimsB64

	// Sign with private key
	privateKeyBytes, err := base64.RawURLEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %v", err)
	}

	// Create ECDSA private key
	curve := elliptic.P256()
	d := new(big.Int).SetBytes(privateKeyBytes)
	privateKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int),
			Y:     new(big.Int),
		},
		D: d,
	}
	privateKey.PublicKey.X, privateKey.PublicKey.Y = curve.ScalarBaseMult(privateKeyBytes)

	// Hash and sign
	hash := sha256.Sum256([]byte(unsignedToken))
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign: %v", err)
	}

	// Convert signature to bytes (r and s are 32 bytes each)
	signature := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):], sBytes)

	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	jwt := unsignedToken + "." + signatureB64

	return "vapid t=" + jwt + ", k=" + publicKeyB64, nil
}

// Helper function to convert uint16 to bytes
func uint16ToBytes(n uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, n)
	return b
}
