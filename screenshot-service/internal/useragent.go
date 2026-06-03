package internal

import "os"

// UserAgent returns the User-Agent used for crawling and preflight requests.
//
// The Chrome token is preserved so target pages still render as they would for
// a real browser, but a ChaseMonitor identifier is appended so site operators
// can see that the request originates from Chase. CHASE_HOSTNAME overrides the
// contact URL, falling back to the project repository.
func UserAgent() string {
	host := os.Getenv("CHASE_HOSTNAME")
	if host == "" {
		host = "https://github.com/NorskHelsenett/chase"
	}
	return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 ChaseMonitor/1.0 (+" + host + ")"
}
