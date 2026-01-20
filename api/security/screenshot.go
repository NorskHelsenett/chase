package security

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ScreenshotService handles communication with the Python screenshot service
type ScreenshotService struct {
	baseURL    string
	httpClient *http.Client
}

// NewScreenshotService creates a new screenshot service client
func NewScreenshotService(baseURL string) *ScreenshotService {
	return &ScreenshotService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: serviceTimeout(),
		},
	}
}

// CaptureScreenshot takes a screenshot of the given URL
func (s *ScreenshotService) CaptureScreenshot(url string) ([]byte, error) {
	baseURL := strings.TrimRight(s.baseURL, "/")
	targetURL := strings.TrimRight(url, "/")
	requestURL := fmt.Sprintf("%s/%s/.png", baseURL, targetURL)
	resp, err := s.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("service returned status %d: %s", resp.StatusCode, string(body))
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if len(imageData) == 0 {
		return nil, fmt.Errorf("screenshot failed: empty response")
	}
	return imageData, nil
}
