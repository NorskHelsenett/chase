package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// CrawlResult contains the extracted HTML and screenshot
type CrawlResult struct {
	URL         string `json:"url"`
	HTML        string `json:"html"`
	Screenshot  string `json:"screenshot"` // base64 encoded PNG
	Error       string `json:"error,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	Epoch       int64  `json:"epoch,omitempty"`
	OGImage     string `json:"og_image,omitempty"`
}

// Crawler handles browser-based web crawling
type Crawler struct {
	browser        *rod.Browser
	consentRemover *ConsentRemover
}

const (
	readyStatePollInterval  = 100 * time.Millisecond
	fontsReadyPollInterval  = 100 * time.Millisecond
	pageSettledStableWindow = 300 * time.Millisecond
	pageSettledPollInterval = 100 * time.Millisecond
	lazyLoadScrollInterval  = 150 * time.Millisecond
	preScreenshotRenderWait = 750 * time.Millisecond
)

// NewCrawler creates a new crawler instance
func NewCrawler() (*Crawler, error) {
	// Find Chrome binary - check common locations
	chromePaths := []string{
		os.Getenv("CHROME_BIN"),         // Environment variable
		"/usr/bin/chromium-browser",     // Alpine Chrome
		"/usr/bin/chromium",             // Generic Chromium
		"/usr/bin/google-chrome",        // Google Chrome
		"/usr/bin/google-chrome-stable", // Google Chrome Stable
	}

	var chromePath string
	for _, path := range chromePaths {
		if path != "" {
			// Check if file exists
			if _, err := os.Stat(path); err == nil {
				chromePath = path
				break
			}
		}
	}

	// Launch browser with rod's launcher
	l := launcher.New().
		Headless(true).
		NoSandbox(true).                                       // Required for Docker
		Set("disable-blink-features", "AutomationControlled"). // Avoid detection
		Set("disable-web-security").                           // Allow cross-origin if needed
		Set("disable-features", "IsolateOrigins,site-per-process").
		Set("disable-dev-shm-usage"). // Overcome limited resource problems
		Set("disable-gpu")            // Disable GPU in headless mode

	// Use found Chrome binary if available, otherwise let rod auto-download
	if chromePath != "" {
		l = l.Bin(chromePath)
	}

	launchURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(launchURL).MustConnect()

	return &Crawler{
		browser:        browser,
		consentRemover: NewConsentRemover(),
	}, nil
}

// Crawl fetches a URL, removes cookie consent, and captures HTML (+ optional screenshot)
func (c *Crawler) Crawl(ctx context.Context, targetURL string, waitTime time.Duration, captureScreenshot bool, fullPageScreenshot bool, viewportWidth int, viewportHeight int) (*CrawlResult, error) {
	result := &CrawlResult{
		URL: targetURL,
	}

	// Create a new page
	page, err := c.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		result.Error = fmt.Sprintf("failed to create page: %v", err)
		return result, err
	}
	defer page.Close()

	// Set context timeout
	page = page.Context(ctx)

	// Set user agent to avoid being blocked
	err = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		AcceptLanguage: "en-US,en;q=0.9",
	})
	if err != nil {
		result.Error = fmt.Sprintf("failed to set user agent: %v", err)
		return result, err
	}

	// Set viewport for consistent screenshots
	if viewportWidth <= 0 {
		viewportWidth = 1920
	}
	if viewportHeight <= 0 {
		viewportHeight = 1080
	}
	err = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             viewportWidth,
		Height:            viewportHeight,
		DeviceScaleFactor: 1,
		Mobile:            false,
	})
	if err != nil {
		result.Error = fmt.Sprintf("failed to set viewport: %v", err)
		return result, err
	}

	// Extract domain for cookie setting
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		result.Error = fmt.Sprintf("failed to parse URL: %v", err)
		return result, err
	}

	// Pre-set consent cookies before navigation (may fail due to browser security, which is fine)
	_ = c.consentRemover.SetConsentCookies(page, parsedURL.Hostname())

	// Navigate to URL
	err = page.Navigate(targetURL)
	if err != nil {
		result.Error = fmt.Sprintf("failed to navigate: %v", err)
		return result, err
	}

	waitTimeout := waitTime
	if waitTimeout <= 0 {
		waitTimeout = 1 * time.Second
	}

	if err := c.waitForRender(page, waitTimeout); err != nil {
		fmt.Printf("Warning: page not fully rendered after %v: %v\n", waitTimeout, err)
	}

	// Remove cookie consent banners (best effort - may not work on all sites)
	_ = c.consentRemover.RemoveConsent(page)

	// Extract metadata before getting HTML
	result.Title = c.extractTitle(page)
	result.Author = c.extractAuthor(page)
	result.PublishedAt = c.extractPublishedDate(page)
	result.OGImage = c.extractOGImage(page)
	result.Description = c.extractDescription(page)

	// Extract full HTML without cleaning to preserve the original document
	htmlResult, err := page.Eval(`() => {
		const doctype = document.doctype;
		let doctypeString = '';
		if (doctype) {
			doctypeString = '<!DOCTYPE ' + doctype.name;
			if (doctype.publicId) {
				doctypeString += ' PUBLIC "' + doctype.publicId + '"';
			}
			if (doctype.systemId) {
				doctypeString += ' "' + doctype.systemId + '"';
			}
			doctypeString += '>';
		}

		return doctypeString + document.documentElement.outerHTML;
	}`)
	if err != nil {
		result.Error = fmt.Sprintf("failed to extract HTML: %v", err)
		return result, err
	}
	result.HTML = htmlResult.Value.Str()

	if captureScreenshot {
		_ = c.triggerLazyLoad(page, waitTimeout)
		_ = c.waitForRender(page, preScreenshotRenderWait)

		_, _ = page.Eval(`() => {
			const style = document.createElement('style');
			style.setAttribute('data-crawler', 'hide-scrollbar');
			style.textContent = '::-webkit-scrollbar { width: 0 !important; height: 0 !important; } * { scrollbar-width: none !important; -ms-overflow-style: none !important; }';
			document.head.appendChild(style);
		}`)

		// Capture screenshot
		quality := 90
		screenshot, err := page.Screenshot(fullPageScreenshot, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormatPng,
			Quality: &quality,
		})
		if err != nil {
			result.Error = fmt.Sprintf("failed to capture screenshot: %v", err)
			return result, err
		}

		// Encode screenshot as base64
		result.Screenshot = base64.StdEncoding.EncodeToString(screenshot)
	}

	return result, nil
}

// Close closes the browser
func (c *Crawler) Close() error {
	if c.browser != nil {
		return c.browser.Close()
	}
	return nil
}

func (c *Crawler) waitForRender(page *rod.Page, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	if err := c.waitForReadyState(page, deadline); err != nil {
		return err
	}
	_ = c.waitForFontsReady(page, deadline)
	if err := c.waitForPageSettled(page, deadline); err != nil {
		return err
	}
	return nil
}

func (c *Crawler) waitForReadyState(page *rod.Page, deadline time.Time) error {
	for time.Now().Before(deadline) {
		state, err := page.Eval(`() => document.readyState`)
		if err == nil && state.Value.Str() == "complete" {
			return nil
		}
		time.Sleep(readyStatePollInterval)
	}
	return fmt.Errorf("readyState not complete")
}

func (c *Crawler) waitForFontsReady(page *rod.Page, deadline time.Time) error {
	for time.Now().Before(deadline) {
		result, err := page.Eval(`() => {
			if (!document.fonts || !document.fonts.status) return true;
			return document.fonts.status === 'loaded';
		}`)
		if err == nil && result.Value.Bool() {
			return nil
		}
		time.Sleep(fontsReadyPollInterval)
	}
	return fmt.Errorf("fonts not loaded")
}

func (c *Crawler) waitForPageSettled(page *rod.Page, deadline time.Time) error {
	var stableSince time.Time
	var lastScrollHeight int
	var lastTotal int
	var lastIncomplete int
	var hasVisible bool

	for time.Now().Before(deadline) {
		result, err := page.Eval(`() => {
			const body = document.body;
			const html = document.documentElement;
			const scrollHeight = Math.max(body?.scrollHeight || 0, html?.scrollHeight || 0);
			const images = Array.from(document.images || []);
			const total = images.length;
			const incomplete = images.filter(img => !img.complete).length;
			const visible = !!(body && body.offsetHeight > 0 && body.offsetWidth > 0);
			return { scrollHeight, total, incomplete, visible };
		}`)
		if err != nil {
			time.Sleep(pageSettledPollInterval)
			continue
		}

		scrollHeight := int(result.Value.Get("scrollHeight").Int())
		total := int(result.Value.Get("total").Int())
		incomplete := int(result.Value.Get("incomplete").Int())
		hasVisible = result.Value.Get("visible").Bool()

		if scrollHeight == lastScrollHeight && total == lastTotal && incomplete == lastIncomplete {
			if !stableSince.IsZero() && time.Since(stableSince) >= pageSettledStableWindow && hasVisible && scrollHeight > 0 && incomplete == 0 {
				return nil
			}
		} else {
			stableSince = time.Now()
		}

		lastScrollHeight = scrollHeight
		lastTotal = total
		lastIncomplete = incomplete
		time.Sleep(pageSettledPollInterval)
	}

	return fmt.Errorf("page not settled")
}

func (c *Crawler) triggerLazyLoad(page *rod.Page, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	lastHeight := -1
	stableBottomCount := 0

	for time.Now().Before(deadline) {
		result, err := page.Eval(`() => {
			const body = document.body;
			const html = document.documentElement;
			const scrollHeight = Math.max(body?.scrollHeight || 0, html?.scrollHeight || 0);
			const scrollTop = window.scrollY || 0;
			const viewport = window.innerHeight || 0;
			return { scrollHeight, scrollTop, viewport };
		}`)
		if err != nil {
			return err
		}

		scrollHeight := int(result.Value.Get("scrollHeight").Int())
		scrollTop := int(result.Value.Get("scrollTop").Int())
		viewport := int(result.Value.Get("viewport").Int())
		atBottom := scrollTop+viewport+2 >= scrollHeight

		if scrollHeight == lastHeight && atBottom {
			stableBottomCount++
			if stableBottomCount >= 2 {
				break
			}
		} else {
			stableBottomCount = 0
		}

		lastHeight = scrollHeight
		_, _ = page.Eval(`() => window.scrollBy(0, window.innerHeight * 0.9)`)
		time.Sleep(lazyLoadScrollInterval)
	}

	_, _ = page.Eval(`() => window.scrollTo(0, 0)`)
	return nil
}

// IsHealthy checks whether the browser connection is still usable.
func (c *Crawler) IsHealthy() error {
	if c == nil || c.browser == nil {
		return fmt.Errorf("browser not initialized")
	}
	_, err := c.browser.Pages()
	return err
}

// extractTitle extracts the page title
func (c *Crawler) extractTitle(page *rod.Page) string {
	// Try OG title first
	if title, err := page.Eval(`() => {
		const og = document.querySelector('meta[property="og:title"]');
		return og ? og.content : '';
	}`); err == nil && title.Value.Str() != "" {
		return title.Value.Str()
	}

	// Fallback to <title> tag
	if title, err := page.Eval(`() => document.title || ''`); err == nil {
		return title.Value.Str()
	}

	return ""
}

// extractDescription extracts the page description
func (c *Crawler) extractDescription(page *rod.Page) string {
	// Try OG description first
	if desc, err := page.Eval(`() => {
		const og = document.querySelector('meta[property="og:description"]');
		return og ? og.content : '';
	}`); err == nil && desc.Value.Str() != "" {
		return desc.Value.Str()
	}

	// Fallback to meta description
	if desc, err := page.Eval(`() => {
		const meta = document.querySelector('meta[name="description"]');
		return meta ? meta.content : '';
	}`); err == nil {
		return desc.Value.Str()
	}

	return ""
}

// extractAuthor extracts the author
func (c *Crawler) extractAuthor(page *rod.Page) string {
	if author, err := page.Eval(`() => {
		// Helper to clean author text
		const cleanAuthor = (text) => {
			if (!text) return '';
			// Remove "By" prefix and normalize whitespace
			return text
				.replace(/^By\s+/i, '')
				.replace(/\s+/g, ' ')
				.trim();
		};

		// Try meta tags first
		const metaSelectors = [
			'meta[property="article:author"]',
			'meta[name="author"]',
			'meta[property="og:article:author"]'
		];
		for (const sel of metaSelectors) {
			const el = document.querySelector(sel);
			if (el && el.content) {
				const cleaned = cleanAuthor(el.content);
				if (cleaned) return cleaned;
			}
		}

		// Try common author elements
		const authorSelectors = [
			'[rel="author"]',
			'.author',
			'[itemprop="author"] [itemprop="name"]',
			'[class*="author"]',
			'[class*="byline"]'
		];
		for (const sel of authorSelectors) {
			const el = document.querySelector(sel);
			if (el && el.textContent) {
				const cleaned = cleanAuthor(el.textContent);
				if (cleaned) return cleaned;
			}
		}

		// Look for "By [Author]" or "Written by [Author]" patterns
		const srLabels = document.querySelectorAll('.sr-only, [class*="sr-only"]');
		for (const label of srLabels) {
			const text = label.textContent.toLowerCase();
			if (text.includes('written by') || text.includes('author')) {
				// Check next sibling or parent's next child
				let sibling = label.nextElementSibling;
				if (!sibling && label.parentElement) {
					sibling = label.parentElement.querySelector('p, span');
				}
				if (sibling) {
					const cleaned = cleanAuthor(sibling.textContent);
					if (cleaned) return cleaned;
				}
			}
		}

		// Look for "By Author" text pattern anywhere
		const bodyText = document.body.innerText;
		const byMatch = bodyText.match(/By\s+([A-Z][a-zA-Z\s]+?)(?:\s*\||$|\n)/);
		if (byMatch) {
			const cleaned = cleanAuthor(byMatch[1]);
			if (cleaned) return cleaned;
		}

		return '';
	}`); err == nil && author.Value.Str() != "" {
		return author.Value.Str()
	}

	return ""
}

// extractPublishedDate extracts the published date
func (c *Crawler) extractPublishedDate(page *rod.Page) string {
	if date, err := page.Eval(`() => {
		// Try meta tags first
		const metaSelectors = [
			'meta[property="article:published_time"]',
			'meta[property="og:article:published_time"]',
			'meta[name="publication_date"]'
		];
		for (const sel of metaSelectors) {
			const el = document.querySelector(sel);
			if (el && el.content) {
				return el.content;
			}
		}

		// Try time elements
		const timeSelectors = [
			'time[itemprop="datePublished"]',
			'time[datetime]',
			'time[pubdate]',
			'.published',
			'[class*="publish"]',
			'[class*="date"]'
		];
		for (const sel of timeSelectors) {
			const el = document.querySelector(sel);
			if (el) {
				const datetime = el.getAttribute('datetime');
				if (datetime) return datetime;
				const text = el.textContent?.trim();
				if (text) return text;
			}
		}

		// Look for "Published on [Date]" patterns with sr-only labels
		const srLabels = document.querySelectorAll('.sr-only, [class*="sr-only"]');
		for (const label of srLabels) {
			const text = label.textContent.toLowerCase();
			if (text.includes('published') || text.includes('date')) {
				// Check next sibling
				let sibling = label.nextElementSibling;
				if (!sibling && label.parentElement) {
					sibling = label.parentElement.querySelector('p, span, time');
				}
				if (sibling) {
					const dateText = sibling.textContent.trim();
					// Basic date pattern check (e.g., "Dec 24, 2025" or "2025-12-24")
					if (dateText.match(/\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|[0-9]{4})/i)) {
						return dateText;
					}
				}
			}
		}

		// Look for date patterns in common locations
		const datePattern = /\b(Jan(?:uary)?|Feb(?:ruary)?|Mar(?:ch)?|Apr(?:il)?|May|Jun(?:e)?|Jul(?:y)?|Aug(?:ust)?|Sep(?:tember)?|Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?)\s+\d{1,2},?\s+\d{4}\b/i;
		const candidates = document.querySelectorAll('p, span, div');
		for (const el of candidates) {
			const text = el.textContent.trim();
			const match = text.match(datePattern);
			if (match && text.length < 100) { // Avoid matching dates in article body
				return match[0];
			}
		}

		return '';
	}`); err == nil && date.Value.Str() != "" {
		return date.Value.Str()
	}

	return ""
}

// extractOGImage extracts the OG image
func (c *Crawler) extractOGImage(page *rod.Page) string {
	if img, err := page.Eval(`() => {
		const og = document.querySelector('meta[property="og:image"]');
		return og ? og.content : '';
	}`); err == nil {
		return img.Value.Str()
	}

	return ""
}
