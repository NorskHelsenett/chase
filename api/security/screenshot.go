package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ScreenshotService handles communication with the Python screenshot service
type ScreenshotService struct {
	baseURL    string
	httpClient *http.Client
}

// ScreenshotRequest represents the parameters we can send to the screenshot service
type ScreenshotRequest struct {
	URL      string `json:"url"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	WaitTime int    `json:"wait_time,omitempty"` // in seconds
	Fullpage bool   `json:"fullpage,omitempty"`
}

// ScreenshotResponse represents the response from the screenshot service
type ScreenshotResponse struct {
	Success   bool   `json:"success"`
	Image     string `json:"image"` // base64 encoded image
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
	Domain    string `json:"url"`
}

// NewScreenshotService creates a new screenshot service client
func NewScreenshotService(baseURL string) *ScreenshotService {
	return &ScreenshotService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CaptureScreenshot takes a screenshot of the given URL
func (s *ScreenshotService) CaptureScreenshot(url string) ([]byte, error) {
	request := ScreenshotRequest{
		URL:      url,
		Width:    1920,
		Height:   1080,
		WaitTime: 3,
		Fullpage: true,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.httpClient.Post(
		fmt.Sprintf("%s/screenshot", s.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ScreenshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("screenshot failed: %s", result.Error)
	}

	return []byte(result.Image), nil
}
