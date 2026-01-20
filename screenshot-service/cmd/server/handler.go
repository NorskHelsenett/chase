package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"bookmarker/crawler/internal"
)

type Handler struct {
	pool        chan *internal.Crawler
	poolSize    int
	poolCreated int
	poolMu      sync.Mutex
}

const defaultCrawlTimeout = 30 * time.Second

func NewHandler() *Handler {
	poolSize := crawlPoolSize()
	if poolSize <= 0 {
		poolSize = 1
	}

	return &Handler{
		pool:     make(chan *internal.Crawler, poolSize),
		poolSize: poolSize,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log request
	start := time.Now()
	var logURL string
	logRequest := true
	defer func() {
		if logRequest {
			log.Printf("[%s] %s %s - %s", r.Method, logURL, r.RemoteAddr, time.Since(start))
		}
	}()

	path := strings.TrimPrefix(r.URL.Path, "/")
	query := r.URL.RawQuery

	// Health check
	if path == "" || path == "health" || path == "healthz" {
		logURL = r.URL.Path
		logRequest = false
		if err := h.ensureCrawlerHealthy(r.Context()); err != nil {
			log.Printf("Crawler unhealthy: %v", err)
			http.Error(w, "Crawler unhealthy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Crawler service running"))
		return
	}

	// Extract URL and format from path
	// Examples: vg.no/.png, https://example.com/.html
	// Format can be in path OR at end of query string (for URLs with ? params)
	var targetURL, format string
	var remainingQuery string

	// Check if format is in the query string (e.g., ?v=123/.md)
	if strings.HasSuffix(query, "/.png") {
		remainingQuery = strings.TrimSuffix(query, "/.png")
		format = "png"
		targetURL = path
	} else if strings.HasSuffix(query, "/.html") {
		remainingQuery = strings.TrimSuffix(query, "/.html")
		format = "html"
		targetURL = path
	} else if strings.HasSuffix(path, "/.png") {
		targetURL = strings.TrimSuffix(path, "/.png")
		format = "png"
		remainingQuery = query
	} else if strings.HasSuffix(path, "/.html") {
		targetURL = strings.TrimSuffix(path, "/.html")
		format = "html"
		remainingQuery = query
	} else {
		logURL = r.URL.Path
		http.Error(w, "Invalid format. Use: /.png or /.html", http.StatusBadRequest)
		return
	}

	if targetURL == "" {
		logURL = r.URL.Path
		http.Error(w, "URL required. Usage: /jonas.grimsgaard.dev/.png or /https://example.com/.json", http.StatusBadRequest)
		return
	}

	// Infer https:// if no scheme provided
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	fullPageScreenshot := false
	viewportWidth := 1920
	viewportHeight := 1080
	if remainingQuery != "" {
		values, err := url.ParseQuery(remainingQuery)
		if err == nil {
			if isTruthyFlag(values, "fullscreen") || isTruthyFlag(values, "fullpage") {
				fullPageScreenshot = true
			}
			values.Del("fullscreen")
			values.Del("fullpage")
			if width, ok := parsePositiveInt(values, "width"); ok {
				viewportWidth = width
				values.Del("width")
			}
			if height, ok := parsePositiveInt(values, "height"); ok {
				viewportHeight = height
				values.Del("height")
			}
			remainingQuery = values.Encode()
		}
	}

	// Append query parameters if present (use remainingQuery after format extraction)
	if remainingQuery != "" {
		targetURL = targetURL + "?" + remainingQuery
	}

	// Set log URL to the target URL being crawled
	logURL = targetURL

	// Create or reuse crawler instance
	ctx, cancel := context.WithTimeout(r.Context(), crawlTimeout())
	defer cancel()

	crawler, err := h.acquireCrawler(ctx)
	if err != nil {
		log.Printf("Failed to create crawler: %v", err)
		http.Error(w, "Failed to initialize crawler", http.StatusInternalServerError)
		return
	}
	var result *internal.CrawlResult
	defer func() {
		h.releaseCrawler(crawler, err == nil && result != nil && result.Error == "")
	}()

	// Crawl the URL
	waitTime := 3 * time.Second
	captureScreenshot := format == "png"
	result, err = crawler.Crawl(ctx, targetURL, waitTime, captureScreenshot, fullPageScreenshot, viewportWidth, viewportHeight)
	if err != nil || result.Error != "" {
		log.Printf("Error crawling %s: %v / %s", targetURL, err, result.Error)
		http.Error(w, "Failed to crawl URL", http.StatusInternalServerError)
		return
	}

	// Respond based on format
	switch format {
	case "png":
		h.servePNG(w, result)
	case "html":
		h.serveHTML(w, result)
	default:
		http.Error(w, "Invalid format", http.StatusBadRequest)
	}
}

func (h *Handler) servePNG(w http.ResponseWriter, result *internal.CrawlResult) {
	screenshotData, err := base64.StdEncoding.DecodeString(result.Screenshot)
	if err != nil {
		log.Printf("Error decoding screenshot: %v", err)
		http.Error(w, "Failed to decode screenshot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
	w.Write(screenshotData)
}

func (h *Handler) serveHTML(w http.ResponseWriter, result *internal.CrawlResult) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
	w.Write([]byte(result.HTML))
}

func isTruthyFlag(values url.Values, key string) bool {
	if _, ok := values[key]; !ok {
		return false
	}

	value := strings.ToLower(strings.TrimSpace(values.Get(key)))
	if value == "" {
		return true
	}

	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parsePositiveInt(values url.Values, key string) (int, bool) {
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		return 0, false
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, false
	}

	return value, true
}

func (h *Handler) Close() error {
	h.poolMu.Lock()
	defer h.poolMu.Unlock()

	close(h.pool)
	for crawler := range h.pool {
		_ = crawler.Close()
	}
	h.poolCreated = 0
	return nil
}

func crawlTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("CRAWL_TIMEOUT"))
	if raw == "" {
		return defaultCrawlTimeout
	}

	if duration, err := time.ParseDuration(raw); err == nil && duration > 0 {
		return duration
	}

	if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	return defaultCrawlTimeout
}

func crawlPoolSize() int {
	raw := strings.TrimSpace(os.Getenv("CRAWLER_POOL_SIZE"))
	if raw == "" {
		return 1
	}

	size, err := strconv.Atoi(raw)
	if err != nil || size <= 0 {
		return 1
	}

	return size
}

func (h *Handler) acquireCrawler(ctx context.Context) (*internal.Crawler, error) {
	select {
	case crawler := <-h.pool:
		return crawler, nil
	default:
	}

	h.poolMu.Lock()
	if h.poolCreated < h.poolSize {
		h.poolCreated++
		h.poolMu.Unlock()

		crawler, err := internal.NewCrawler()
		if err != nil {
			h.poolMu.Lock()
			h.poolCreated--
			h.poolMu.Unlock()
			return nil, err
		}
		return crawler, nil
	}
	h.poolMu.Unlock()

	select {
	case crawler := <-h.pool:
		return crawler, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (h *Handler) releaseCrawler(crawler *internal.Crawler, healthy bool) {
	if crawler == nil {
		return
	}

	if !healthy {
		_ = crawler.Close()
		h.poolMu.Lock()
		if h.poolCreated > 0 {
			h.poolCreated--
		}
		h.poolMu.Unlock()
		return
	}

	select {
	case h.pool <- crawler:
	default:
		_ = crawler.Close()
		h.poolMu.Lock()
		if h.poolCreated > 0 {
			h.poolCreated--
		}
		h.poolMu.Unlock()
	}
}

func (h *Handler) ensureCrawlerHealthy(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	crawler, err := h.acquireCrawler(ctx)
	if err != nil {
		return err
	}

	healthy := true
	if err := crawler.IsHealthy(); err != nil {
		healthy = false
	}

	h.releaseCrawler(crawler, healthy)
	if !healthy {
		return fmt.Errorf("crawler unhealthy")
	}
	return nil
}
