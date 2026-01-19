package servers

import (
	"errors"
	"net/url"
	"strings"

	"github.com/norskhelsenett/chase/utils"
)

func normalizeServerURL(rawURL string) (string, error) {
	formattedURL, err := utils.EnsureHTTPS(rawURL)
	if err != nil {
		return "", err
	}

	parsedURL, err := url.Parse(formattedURL)
	if err != nil {
		return "", err
	}

	if parsedURL.Host == "" {
		return "", errors.New("url must contain a valid host")
	}

	cleanURL := strings.TrimPrefix(strings.TrimPrefix(formattedURL, "https://"), "http://")
	return cleanURL, nil
}
