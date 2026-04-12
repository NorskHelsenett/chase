package servers

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
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
	"gorm.io/gorm"
)

const faviconFetchTimeout = 8 * time.Second
const maxFaviconBytes = 1024 * 1024

func GetServerFavicon(c *gin.Context) {
	db := database.GetDB()

	var server Server
	if err := db.First(&server, c.Param("id")).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	if len(server.FaviconData) > 0 {
		serveCachedFavicon(c, server)
		return
	}

	if err := refreshServerFavicon(db, &server); err == nil && len(server.FaviconData) > 0 {
		serveCachedFavicon(c, server)
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

func refreshServerFavicon(db *gorm.DB, server *Server) error {
	if server == nil {
		return fmt.Errorf("server is nil")
	}

	icon, err := fetchBestFavicon(*server)
	if err != nil {
		return err
	}

	now := time.Now()
	server.Favicon = icon.SourceURL
	server.FaviconMime = icon.ContentType
	server.FaviconData = icon.Data
	server.FaviconFetchedAt = &now

	return db.Model(server).Updates(map[string]any{
		"favicon":            server.Favicon,
		"favicon_mime":       server.FaviconMime,
		"favicon_data":       server.FaviconData,
		"favicon_fetched_at": server.FaviconFetchedAt,
	}).Error
}

type cachedFavicon struct {
	SourceURL   string
	ContentType string
	Data        []byte
}

func fetchBestFavicon(server Server) (*cachedFavicon, error) {
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

		data, err := io.ReadAll(io.LimitReader(resp.Body, maxFaviconBytes+1))
		resp.Body.Close()
		if err != nil {
			continue
		}
		if len(data) == 0 || len(data) > maxFaviconBytes {
			continue
		}

		return &cachedFavicon{
			SourceURL:   faviconURL,
			ContentType: contentType,
			Data:        data,
		}, nil
	}

	return nil, fmt.Errorf("no favicon available")
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

func serveCachedFavicon(c *gin.Context, server Server) {
	etag := serverFaviconETag(server.FaviconData)
	if match := strings.TrimSpace(c.GetHeader("If-None-Match")); match != "" && match == etag {
		c.Status(http.StatusNotModified)
		return
	}

	contentType := server.FaviconMime
	if strings.TrimSpace(contentType) == "" {
		contentType = "image/x-icon"
	}

	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("ETag", etag)
	c.Header("Content-Type", contentType)
	modTime := server.UpdatedAt
	if server.FaviconFetchedAt != nil {
		modTime = *server.FaviconFetchedAt
		c.Header("Last-Modified", server.FaviconFetchedAt.UTC().Format(http.TimeFormat))
	}
	http.ServeContent(c.Writer, c.Request, "", modTime, bytes.NewReader(server.FaviconData))
}

func serverFaviconETag(data []byte) string {
	sum := sha256.Sum256(data)
	return `"` + hex.EncodeToString(sum[:]) + `"`
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
