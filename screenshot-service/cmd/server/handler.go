package main

import (
	"context"
	"encoding/base64"
	"errors"
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
	httpClient  *http.Client
}

const defaultCrawlTimeout = 10 * time.Second
const defaultPreflightTimeout = 2 * time.Second
const defaultQueueTimeout = 500 * time.Millisecond

func NewHandler() *Handler {
	poolSize := crawlPoolSize()
	if poolSize <= 0 {
		poolSize = 1
	}

	return &Handler{
		pool:       make(chan *internal.Crawler, poolSize),
		poolSize:   poolSize,
		httpClient: &http.Client{Timeout: preflightTimeout()},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log request
	start := time.Now()
	logURL := r.URL.String()
	logRequest := true
	recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
	defer func() {
		if logRequest {
			log.Printf("[%s] %s %s - %d %s", r.Method, logURL, r.RemoteAddr, recorder.statusCode, time.Since(start))
		}
	}()

	w = recorder

	path := strings.TrimPrefix(r.URL.Path, "/")
	query := r.URL.RawQuery

	// Health check
	if path == "" || path == "health" || path == "healthz" {
		logURL = r.URL.String()
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
		logURL = r.URL.String()
		http.Error(w, "Invalid format. Use: /.png or /.html", http.StatusBadRequest)
		return
	}

	if targetURL == "" {
		logURL = r.URL.String()
		http.Error(w, "URL required. Usage: /jonas.grimsgaard.dev/.png or /https://example.com/.json", http.StatusBadRequest)
		return
	}

	// Infer https:// if no scheme provided
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	if format == "html" && isLikelyNonHTML(targetURL) {
		http.Error(w, "Target is not HTML", http.StatusUnsupportedMediaType)
		return
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

	// Keep log URL as the full request path for clarity.

	// Preflight to get upstream status before launching a browser.
	preflightStatusCode, err := h.preflightStatus(targetURL)
	if err != nil {
		log.Printf("Preflight failed %s (%s)", targetURL, format)
		http.Error(w, "Failed to reach URL", statusFromCrawlError(err, "preflight failed"))
		return
	}
	if preflightStatusCode >= 400 {
		log.Printf("Preflight status %s (%s): %d", targetURL, format, preflightStatusCode)
		http.Error(w, http.StatusText(preflightStatusCode), preflightStatusCode)
		return
	}

	// Create or reuse crawler instance
	ctx, cancel := context.WithTimeout(r.Context(), crawlTimeout())
	defer cancel()

	queueCtx, queueCancel := withQueueTimeout(ctx)
	defer queueCancel()

	crawler, err := h.acquireCrawler(queueCtx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			w.Header().Set("Retry-After", retryAfterSeconds())
			http.Error(w, "Crawler busy", http.StatusServiceUnavailable)
			return
		}
		if errors.Is(err, context.Canceled) {
			http.Error(w, "Request canceled", http.StatusRequestTimeout)
			return
		}
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
		log.Printf("Error crawling %s (%s): %v / %s", targetURL, format, err, result.Error)
		http.Error(w, "Failed to crawl URL", statusFromCrawlError(err, result.Error))
		return
	}

	statusCode := http.StatusOK
	if result.StatusCode >= 100 && result.StatusCode <= 599 {
		statusCode = result.StatusCode
	}

	// Respond based on format
	switch format {
	case "png":
		h.servePNG(w, result, statusCode)
	case "html":
		h.serveHTML(w, result, statusCode)
	default:
		http.Error(w, "Invalid format", http.StatusBadRequest)
	}
}

func (h *Handler) servePNG(w http.ResponseWriter, result *internal.CrawlResult, statusCode int) {
	screenshotData, err := base64.StdEncoding.DecodeString(result.Screenshot)
	if err != nil {
		log.Printf("Error decoding screenshot: %v", err)
		http.Error(w, "Failed to decode screenshot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
	w.WriteHeader(statusCode)
	w.Write(screenshotData)
}

func (h *Handler) serveHTML(w http.ResponseWriter, result *internal.CrawlResult, statusCode int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
	w.WriteHeader(statusCode)
	w.Write([]byte(result.HTML))
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
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

func isLikelyNonHTML(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	path := strings.ToLower(parsed.Path)
	segments := strings.Split(path, "/")
	for _, ext := range []string{
		".js",
		".css",
		".json",
		".xml",
		".png",
		".jpg",
		".jpeg",
		".gif",
		".webp",
		".svg",
		".ico",
		".woff",
		".woff2",
		".ttf",
		".otf",
		".eot",
		".pdf",
		".zip",
		".gz",
		".tgz",
		".rar",
		".7z",
		".mp4",
		".mp3",
		".avi",
		".mov",
		".m4a",
	} {
		if strings.HasSuffix(path, ext) {
			return true
		}
		for _, segment := range segments {
			if strings.HasSuffix(segment, ext) {
				return true
			}
		}
	}
	return false
}

func statusFromCrawlError(err error, resultError string) int {
	errText := strings.ToLower(fmt.Sprintf("%v %s", err, resultError))
	switch {
	case strings.Contains(errText, "context deadline exceeded"):
		return http.StatusGatewayTimeout
	case strings.Contains(errText, "err_address_unreachable"):
		return http.StatusBadGateway
	case strings.Contains(errText, "err_name_not_resolved"):
		return http.StatusBadGateway
	case strings.Contains(errText, "err_connection_refused"):
		return http.StatusBadGateway
	case strings.Contains(errText, "err_timed_out"):
		return http.StatusGatewayTimeout
	case strings.Contains(errText, "net::err"):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
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

func preflightTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("PREFLIGHT_TIMEOUT_MS"))
	if raw == "" {
		return defaultPreflightTimeout
	}

	if millis, err := strconv.Atoi(raw); err == nil && millis > 0 {
		return time.Duration(millis) * time.Millisecond
	}

	return defaultPreflightTimeout
}

func queueTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("CRAWLER_QUEUE_TIMEOUT_MS"))
	if raw == "" {
		return defaultQueueTimeout
	}

	if millis, err := strconv.Atoi(raw); err == nil && millis >= 0 {
		return time.Duration(millis) * time.Millisecond
	}

	return defaultQueueTimeout
}

func withQueueTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := queueTimeout()
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

func retryAfterSeconds() string {
	seconds := int(crawlTimeout().Seconds())
	if seconds <= 0 {
		seconds = int(defaultCrawlTimeout.Seconds())
	}
	return strconv.Itoa(seconds)
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

func (h *Handler) preflightStatus(targetURL string) (int, error) {
	req, err := http.NewRequest(http.MethodHead, targetURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()

	return resp.StatusCode, nil
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
