package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/norskhelsenett/chase/database"
	"github.com/norskhelsenett/chase/types"
	"github.com/norskhelsenett/chase/utils"
)

// serversNeedingScreenshot returns the active servers to capture screenshots for.
// When missingOnly is true, servers that already have a usable screenshot — a
// real image (not a failure marker), non-empty, and within the cache TTL — are
// excluded, leaving only those the grid currently can't show.
func serversNeedingScreenshot(missingOnly bool) ([]types.Server, error) {
	db := database.GetDB()

	var servers []types.Server
	if err := db.Where("active = ?", true).Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch active servers: %w", err)
	}

	if !missingOnly {
		return servers, nil
	}

	// Set of server URLs that already have a usable screenshot.
	var rows []struct{ ServerURL string }
	if err := db.Model(&Screenshot{}).
		Where("mime_type NOT LIKE 'error/%'").
		Where("octet_length(data) > 0").
		Where("created_at > ?", time.Now().Add(-screenshotCacheTTL)).
		Select("server_url").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch existing screenshots: %w", err)
	}

	have := make(map[string]bool, len(rows))
	for _, r := range rows {
		have[r.ServerURL] = true
	}

	missing := make([]types.Server, 0)
	for _, s := range servers {
		if !have[utils.StripProtocol(s.URL)] {
			missing = append(missing, s)
		}
	}
	return missing, nil
}

// runScreenshotCapture captures screenshots for the given servers using a
// bounded worker pool, reporting "<done>/<total>" progress. Individual
// failures are counted but never abort the run — the capture layer records a
// failure marker for those so they aren't retried in a tight loop.
func runScreenshotCapture(ctx context.Context, progress func(string), servers []types.Server) (string, error) {
	total := len(servers)
	if total == 0 {
		return "no servers need a screenshot", nil
	}

	progress(fmt.Sprintf("0/%d", total))

	workers := getBatchWorkerCount()
	if workers > total {
		workers = total
	}

	var (
		mu     sync.Mutex
		done   int
		ok     int
		failed int
	)

	serverChan := make(chan types.Server)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for srv := range serverChan {
				if ctx.Err() != nil {
					return
				}
				// captureAndSendScreenshot stores the result; nil context means
				// it captures and persists without writing an HTTP response.
				err := captureAndSendScreenshot(nil, srv.URL, false, 0)

				mu.Lock()
				done++
				if err != nil {
					failed++
				} else {
					ok++
				}
				progress(fmt.Sprintf("%d/%d", done, total))
				mu.Unlock()
			}
		}()
	}

sendLoop:
	for _, srv := range servers {
		select {
		case <-ctx.Done():
			break sendLoop
		case serverChan <- srv:
		}
	}
	close(serverChan)
	wg.Wait()

	if ctx.Err() != nil {
		return fmt.Sprintf("cancelled — %d/%d captured, %d failed", ok, total, failed), ctx.Err()
	}
	return fmt.Sprintf("captured %d/%d screenshots — %d failed", ok, total, failed), nil
}

// RunScreenshotMissing captures screenshots only for active servers that don't
// currently have a usable (fresh, non-error, non-empty) screenshot. Designed to
// be called from the scheduler as a manually-triggered job.
func RunScreenshotMissing(ctx context.Context, progress func(string)) (string, error) {
	servers, err := serversNeedingScreenshot(true)
	if err != nil {
		return "", err
	}
	return runScreenshotCapture(ctx, progress, servers)
}

// RunScreenshotRefresh (re)captures screenshots for all active servers,
// refreshing existing ones regardless of age. Designed to be called from the
// scheduler as a manually-triggered job.
func RunScreenshotRefresh(ctx context.Context, progress func(string)) (string, error) {
	servers, err := serversNeedingScreenshot(false)
	if err != nil {
		return "", err
	}
	return runScreenshotCapture(ctx, progress, servers)
}
