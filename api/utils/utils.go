package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
)

func GetEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

func GenerateAPIToken() string {
	b := make([]byte, 32) // 32 bytes will give us more than enough characters
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Error generating random bytes: %v", err)
	}

	// Use RawURLEncoding to avoid padding characters
	encoded := base64.RawURLEncoding.EncodeToString(b)

	// Remove any non-alphanumeric characters
	alphanumeric := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, encoded)

	// Trim to ensure consistent length and add prefix
	return "sk-" + alphanumeric[:32]
}

// ParseStringToInt converts a string to an integer
// Returns the integer value and any error that occurred during parsing
func ParseStringToInt(s string) (int, error) {
	// Remove any non-numeric characters
	s = strings.TrimSpace(s)

	// Parse the string as an integer
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
