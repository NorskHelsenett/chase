package security

import (
	"os"
	"strconv"
	"time"
)

const defaultServiceTimeout = 3 * time.Minute

func serviceTimeout() time.Duration {
	raw := os.Getenv("SCREENSHOT_SERVICE_TIMEOUT_SECONDS")
	if raw == "" {
		return defaultServiceTimeout
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return defaultServiceTimeout
	}
	return time.Duration(seconds) * time.Second
}
