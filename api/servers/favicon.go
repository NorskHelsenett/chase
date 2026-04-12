package servers

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
)

const faviconFetchTimeout = 8 * time.Second

func GetServerFavicon(c *gin.Context) {
	db := database.GetDB()

	var server Server
	if err := db.First(&server, c.Param("id")).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	for _, faviconURL := range faviconCandidates(server) {
		resp, err := fetchFavicon(server, faviconURL)
		if err != nil {
			continue
		}

		contentType := resp.Header.Get("Content-Type")
		if !isAllowedFaviconContentType(contentType) {
			resp.Body.Close()
			continue
		}
		if strings.TrimSpace(contentType) == "" {
			contentType = "image/x-icon"
		}

		c.Header("Cache-Control", "private, max-age=3600")
		c.Header("Content-Type", contentType)
		if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
			c.Header("Last-Modified", lastModified)
		}
		if etag := resp.Header.Get("ETag"); etag != "" {
			c.Header("ETag", etag)
		}

		_, _ = io.Copy(c.Writer, resp.Body)
		resp.Body.Close()
		return
	}

	c.Status(http.StatusNotFound)
}

func faviconCandidates(server Server) []string {
	seen := make(map[string]struct{})
	var candidates []string

	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		if _, ok := seen[raw]; ok {
			return
		}
		seen[raw] = struct{}{}
		candidates = append(candidates, raw)
	}

	primary := resolveServerURL(server, server.Favicon)
	add(primary)

	if primary != "" {
		if parsed, err := url.Parse(primary); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			rootIcon := *parsed
			rootIcon.Path = "/favicon.ico"
			rootIcon.RawQuery = ""
			rootIcon.Fragment = ""
			add(rootIcon.String())
		}
	}

	add(resolveServerURL(server, "/favicon.ico"))

	return candidates
}

func resolveServerURL(server Server, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err == nil && parsed.IsAbs() {
		return parsed.String()
	}

	base, err := url.Parse("https://" + server.URL)
	if err != nil {
		return ""
	}

	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}

	ref, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	return base.ResolveReference(ref).String()
}

func fetchFavicon(server Server, targetURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

	scannerURL := os.Getenv("CHASE_HOSTNAME")
	if scannerURL == "" {
		scannerURL = "https://github.com/NorskHelsenett/chase"
	}

	req.Header.Set("User-Agent", "ChaseMonitor/1.0 (+"+scannerURL+") Automated Security Scanner for "+server.URL)
	req.Header.Set("Accept", "image/x-icon,image/vnd.microsoft.icon,image/png,image/svg+xml,image/*;q=0.8,*/*;q=0.5")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: server.AllowInsecure,
			},
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
		},
		Timeout: faviconFetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !server.FollowRedirect {
				return http.ErrUseLastResponse
			}
			if len(via) >= 10 {
				return fmt.Errorf("stopped after %d redirects", len(via))
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected favicon status: %d", resp.StatusCode)
	}

	return resp, nil
}

func isAllowedFaviconContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if strings.HasPrefix(contentType, "image/") {
		return true
	}

	switch contentType {
	case "application/octet-stream":
		return true
	case "":
		return true
	default:
		return false
	}
}
