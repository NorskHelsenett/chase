package utils

import (
	"fmt"
	"net/url"
	"strings"
)

func StripProtocol(url string) string {
	if idx := strings.Index(url, "://"); idx >= 0 {
		return url[idx+3:]
	}
	return url
}

func EnsureHTTPS(inputURL string) (string, error) {
	// Trim any whitespace
	inputURL = strings.TrimSpace(inputURL)

	// Check if empty
	if inputURL == "" {
		return "", fmt.Errorf("empty URL provided")
	}

	// If it already has a scheme, validate it
	if strings.Contains(inputURL, "://") {
		parsedURL, err := url.Parse(inputURL)
		if err != nil {
			return "", fmt.Errorf("invalid URL format: %w", err)
		}

		// If it's http, upgrade to https
		if parsedURL.Scheme == "http" {
			parsedURL.Scheme = "https"
			return parsedURL.String(), nil
		}

		// If it's already https, return as is
		if parsedURL.Scheme == "https" {
			return inputURL, nil
		}

		// Don't modify other schemes
		return "", fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	// No scheme present, add https://
	withPrefix := "https://" + inputURL

	// Validate the final URL
	_, err := url.Parse(withPrefix)
	if err != nil {
		return "", fmt.Errorf("invalid URL even after adding https prefix: %w", err)
	}

	return withPrefix, nil
}
