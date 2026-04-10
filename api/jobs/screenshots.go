package jobs

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/norskhelsenett/chase/database"
	"gorm.io/gorm"
)

// RegisterScreenshotJobs registers the screenshot-related jobs.
// captureFunc is the function that captures and stores a screenshot by domain.
func RegisterScreenshotJobs(captureFunc func(domain string) error) {
	Register(
		"backfill-screenshots",
		"Capture screenshots for servers that are missing one",
		"every 6h",
		func(ctx context.Context, appendLog func(string)) (int, int, int, error) {
			return runBackfillScreenshots(ctx, appendLog, captureFunc)
		},
	)
}

func runBackfillScreenshots(ctx context.Context, appendLog func(string), captureFunc func(domain string) error) (total, completed, failed int, err error) {
	db := database.GetDB()

	// Get all active server URLs
	var serverURLs []string
	if err := db.Table("servers").
		Where("active = ? AND deleted_at IS NULL", true).
		Pluck("url", &serverURLs).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("failed to query servers: %w", err)
	}

	if len(serverURLs) == 0 {
		appendLog("No active servers found")
		return 0, 0, 0, nil
	}

	// Get server URLs that already have a valid screenshot
	var screenshottedURLs []string
	if err := db.Table("screenshots").
		Where("mime_type = ? AND data IS NOT NULL AND length(data) > 0", "image/png").
		Pluck("server_url", &screenshottedURLs).Error; err != nil && err != gorm.ErrRecordNotFound {
		return 0, 0, 0, fmt.Errorf("failed to query screenshots: %w", err)
	}

	screenshottedSet := make(map[string]bool, len(screenshottedURLs))
	for _, u := range screenshottedURLs {
		screenshottedSet[strings.ToLower(u)] = true
	}

	// Find servers missing screenshots
	var missing []string
	for _, u := range serverURLs {
		normalized := strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(u), "https://"), "http://")
		if !screenshottedSet[normalized] {
			missing = append(missing, u)
		}
	}

	total = len(missing)
	if total == 0 {
		appendLog("All servers already have screenshots")
		return 0, 0, 0, nil
	}

	appendLog(fmt.Sprintf("Found %d servers missing screenshots", total))

	for _, serverURL := range missing {
		select {
		case <-ctx.Done():
			appendLog("Job cancelled")
			return total, completed, failed, nil
		default:
		}

		domain := serverURL
		if !strings.HasPrefix(domain, "http") {
			domain = "https://" + domain
		}

		appendLog(fmt.Sprintf("Capturing screenshot for %s", serverURL))
		if captureErr := captureFunc(domain); captureErr != nil {
			failed++
			appendLog(fmt.Sprintf("Failed: %s - %v", serverURL, captureErr))
		} else {
			completed++
			appendLog(fmt.Sprintf("Done: %s", serverURL))
		}
	}

	appendLog(fmt.Sprintf("Finished: %d/%d successful, %d failed", completed, total, failed))
	return total, completed, failed, nil
}

// TriggerScreenshot captures a screenshot for a server URL in the background.
// Called when a new server is added or a security scan completes.
func TriggerScreenshot(serverURL string, captureFunc func(domain string) error) {
	domain := serverURL
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}
	go func() {
		if err := captureFunc(domain); err != nil {
			log.Printf("[jobs] Screenshot capture failed for %s: %v", serverURL, err)
		} else {
			log.Printf("[jobs] Screenshot captured for %s", serverURL)
		}
	}()
}
