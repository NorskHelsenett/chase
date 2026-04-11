package servers

import (
	"context"
	"fmt"
	"log"

	"github.com/norskhelsenett/chase/database"
)

// RecheckInactiveServers pings all deactivated servers to see if they've come back online.
// If a server responds with its expected status code, it is reactivated.
func RecheckInactiveServers(ctx context.Context, progress func(string)) (string, error) {
	db := database.GetDB()

	var inactive []Server
	if err := db.Where("active = ?", false).Find(&inactive).Error; err != nil {
		return "", fmt.Errorf("failed to fetch inactive servers: %w", err)
	}

	if len(inactive) == 0 {
		return "no inactive servers to check", nil
	}

	progress(fmt.Sprintf("checking %d inactive servers", len(inactive)))

	reactivated := 0
	checked := 0
	for _, srv := range inactive {
		if ctx.Err() != nil {
			return fmt.Sprintf("cancelled after %d/%d checked, %d reactivated", checked, len(inactive), reactivated), ctx.Err()
		}

		checked++
		progress(fmt.Sprintf("%d/%d — checking %s", checked, len(inactive), srv.URL))

		result := pingServer(srv)

		if result.Error == "" && result.StatusCode == srv.ExpectedStatusCode {
			srv.Active = true
			srv.Comment = ""
			if err := db.Save(&srv).Error; err != nil {
				log.Printf("Failed to reactivate server %s: %v", srv.URL, err)
				continue
			}
			reactivated++
			log.Printf("Reactivated server %s (status %d)", srv.URL, result.StatusCode)
		}
	}

	return fmt.Sprintf("checked %d inactive servers, reactivated %d", len(inactive), reactivated), nil
}
