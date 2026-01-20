package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
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
	// YouTube-specific fields
	ViewCount       string `json:"view_count,omitempty"`
	LikeCount       string `json:"like_count,omitempty"`
	Duration        int    `json:"duration,omitempty"`
	DurationString  string `json:"duration_string,omitempty"`
	ChannelName     string `json:"channel_name,omitempty"`
	ChannelURL      string `json:"channel_url,omitempty"`
	ChannelAvatar   string `json:"channel_avatar,omitempty"`
	SubscriberCount string `json:"subscriber_count,omitempty"`
}

// Crawler handles browser-based web crawling
type Crawler struct {
	browser        *rod.Browser
	consentRemover *ConsentRemover
}

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
		waitTimeout = 2 * time.Second
	}

	if err := c.waitForRender(page, waitTimeout); err != nil {
		fmt.Printf("Warning: page not fully rendered after %v: %v\n", waitTimeout, err)
	}

	// Remove cookie consent banners (best effort - may not work on all sites)
	_ = c.consentRemover.RemoveConsent(page)

	// Check if this is a YouTube video and expand content
	if isYouTubePage(page) {
		// Click the "...more" button to expand description if it exists
		_, _ = page.Eval(`() => {
			const expandButton = document.querySelector('tp-yt-paper-button#expand, #expand, button[aria-label*="more"]');
			if (expandButton) {
				expandButton.click();
			}
		}`)
		_ = c.waitForRender(page, 1*time.Second)
	}

	// Extract metadata before getting HTML
	result.Title = c.extractTitle(page)
	result.Author = c.extractAuthor(page)
	result.PublishedAt = c.extractPublishedDate(page)
	result.OGImage = c.extractOGImage(page)

	// Check if this is a YouTube video - extract description differently
	if isYouTubePage(page) {
		// Extract full description from expanded content
		if desc, err := page.Eval(`() => {
			// Try to get the expanded description content
			const descEl = document.querySelector('ytd-text-inline-expander #content, #description-inline-expander #content, #description');
			if (descEl) {
				return descEl.textContent?.trim() || '';
			}
			// Fallback to meta description
			const meta = document.querySelector('meta[property="og:description"]');
			return meta ? meta.content : '';
		}`); err == nil && desc.Value.Str() != "" {
			result.Description = desc.Value.Str()
		} else {
			result.Description = c.extractDescription(page)
		}
		c.extractYouTubeMetadata(page, result)
	} else {
		result.Description = c.extractDescription(page)
	}

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
		_ = c.waitForRender(page, 1*time.Second)

		_, _ = page.Eval(`() => {
			const style = document.createElement('style');
			style.setAttribute('data-crawler', 'hide-scrollbar');
			style.textContent = '::-webkit-scrollbar { width: 0 !important; height: 0 !important; } * { scrollbar-width: none !important; -ms-overflow-style: none !important; }';
			document.head.appendChild(style);
		}`)

		screenshot, err := c.captureScreenshotWithRetry(page, fullPageScreenshot)
		if err != nil {
			result.Error = fmt.Sprintf("failed to capture screenshot: %v", err)
			return result, err
		}

		// Encode screenshot as base64
		result.Screenshot = base64.StdEncoding.EncodeToString(screenshot)
	}

	return result, nil
}

func (c *Crawler) captureScreenshotWithRetry(page *rod.Page, fullPage bool) ([]byte, error) {
	quality := 90
	screenshot, err := page.Screenshot(fullPage, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatPng,
		Quality: &quality,
	})
	if err == nil {
		return screenshot, nil
	}

	time.Sleep(500 * time.Millisecond)
	_ = c.waitForRender(page, 1*time.Second)
	return page.Screenshot(fullPage, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatPng,
		Quality: &quality,
	})
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
		time.Sleep(100 * time.Millisecond)
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
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("fonts not loaded")
}

func (c *Crawler) waitForPageSettled(page *rod.Page, deadline time.Time) error {
	const stableWindow = 300 * time.Millisecond
	const pollInterval = 100 * time.Millisecond

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
			time.Sleep(pollInterval)
			continue
		}

		scrollHeight := int(result.Value.Get("scrollHeight").Int())
		total := int(result.Value.Get("total").Int())
		incomplete := int(result.Value.Get("incomplete").Int())
		hasVisible = result.Value.Get("visible").Bool()

		if scrollHeight == lastScrollHeight && total == lastTotal && incomplete == lastIncomplete {
			if !stableSince.IsZero() && time.Since(stableSince) >= stableWindow && hasVisible && scrollHeight > 0 && incomplete == 0 {
				return nil
			}
		} else {
			stableSince = time.Now()
		}

		lastScrollHeight = scrollHeight
		lastTotal = total
		lastIncomplete = incomplete
		time.Sleep(pollInterval)
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
		time.Sleep(150 * time.Millisecond)
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

// extractYouTubeMetadata extracts YouTube-specific metadata
func (c *Crawler) extractYouTubeMetadata(page *rod.Page, result *CrawlResult) {
	// Extract view count and publish date from the info element
	if info, err := page.Eval(`() => {
		const infoEl = document.querySelector('yt-formatted-string#info');
		if (!infoEl) return { views: '', date: '' };
		
		const spans = infoEl.querySelectorAll('span[dir="auto"]');
		const views = spans[0]?.textContent?.trim() || '';
		const date = spans[2]?.textContent?.trim() || '';
		
		return { views, date };
	}`); err == nil && info != nil {
		viewsVal := info.Value.Get("views").Str()
		if viewsVal != "" {
			// Remove " views" suffix and other text, keep just the number
			viewsVal = strings.TrimSuffix(viewsVal, " views")
			viewsVal = strings.TrimSuffix(viewsVal, " view")
			viewsVal = strings.TrimSpace(viewsVal)
			// Remove commas from numbers
			viewsVal = strings.ReplaceAll(viewsVal, ",", "")
			result.ViewCount = viewsVal
		}
		// Always use YouTube's date from the info element (more accurate)
		dateVal := info.Value.Get("date").Str()
		if dateVal != "" {
			result.Epoch = c.parseYouTubeEpoch(dateVal)
			// Try to normalize the date to ISO format
			if normalizedDate := c.normalizeYouTubeDate(dateVal); normalizedDate != "" {
				result.PublishedAt = normalizedDate
			} else {
				result.PublishedAt = dateVal
			}
		}
	}

	// Extract like count from the like button
	if likes, err := page.Eval(`() => {
		// Try to find like button with aria-label containing like count
		const likeButton = document.querySelector('like-button-view-model button[aria-label*="like"]');
		if (likeButton) {
			const ariaLabel = likeButton.getAttribute('aria-label') || '';
			// Extract number from aria-label like "like this video along with 16,574 other people"
			const match = ariaLabel.match(/([0-9,]+)/);
			if (match) {
				return match[1];
			}
			// Fallback: get text content (like "16K")
			const textContent = likeButton.querySelector('.yt-spec-button-shape-next__button-text-content');
			if (textContent) {
				return textContent.textContent.trim();
			}
		}
		// Alternative selectors
		const altLikeButton = document.querySelector('button[aria-label*="like this video"]');
		if (altLikeButton) {
			const text = altLikeButton.querySelector('.yt-spec-button-shape-next__button-text-content, #text');
			if (text) return text.textContent.trim();
		}
		return '';
	}`); err == nil && likes != nil && likes.Value.Str() != "" {
		likesVal := likes.Value.Str()
		// Remove commas from numbers
		likesVal = strings.ReplaceAll(likesVal, ",", "")
		result.LikeCount = likesVal
	}

	// Extract video duration from the player or meta tags
	if duration, err := page.Eval(`() => {
		// Try to get duration from meta tag (ISO 8601 format like PT38M18S)
		const metaDuration = document.querySelector('meta[itemprop="duration"]');
		if (metaDuration && metaDuration.content) {
			// Parse ISO 8601 duration (PT38M18S -> 38:18)
			const iso = metaDuration.content;
			const match = iso.match(/PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?/);
			if (match) {
				const hours = match[1] || '0';
				const minutes = match[2] || '0';
				const seconds = match[3] || '0';
				
				if (hours !== '0') {
					return hours + ':' + minutes.padStart(2, '0') + ':' + seconds.padStart(2, '0');
				} else {
					return minutes + ':' + seconds.padStart(2, '0');
				}
			}
		}
		
		// Fallback: try to find duration in the page
		const durationEl = document.querySelector('.ytp-time-duration, .ytd-thumbnail-overlay-time-status-renderer');
		if (durationEl) {
			return durationEl.textContent.trim();
		}
		
		return '';
	}`); err == nil && duration != nil && duration.Value.Str() != "" {
		result.DurationString = duration.Value.Str()
		result.Duration = parseDurationSeconds(result.DurationString)
	}

	// Extract channel name and URL
	if channel, err := page.Eval(`() => {
		// Try the channel link in the video info
		const channelLink = document.querySelector('ytd-channel-name a, #owner a, #upload-info a');
		if (channelLink) {
			return {
				name: channelLink.textContent?.trim() || '',
				url: channelLink.href || ''
			};
		}
		
		// Fallback to meta tags
		const channelMeta = document.querySelector('link[itemprop="name"]');
		const channelURLMeta = document.querySelector('link[itemprop="url"]');
		return {
			name: channelMeta?.content || '',
			url: channelURLMeta?.href || ''
		};
	}`); err == nil && channel != nil {
		nameVal := channel.Value.Get("name").Str()
		if nameVal != "" {
			result.ChannelName = nameVal
			// Always use channel name as author for YouTube videos
			result.Author = nameVal
		}
		urlVal := channel.Value.Get("url").Str()
		if urlVal != "" {
			result.ChannelURL = urlVal
		}
	}

	// Extract channel avatar
	if avatar, err := page.Eval(`() => {
		// Try to find the avatar image in the owner renderer
		const avatarImg = document.querySelector('ytd-video-owner-renderer yt-img-shadow#avatar img, #avatar img');
		if (avatarImg && avatarImg.src) {
			return avatarImg.src;
		}
		return '';
	}`); err == nil && avatar != nil && avatar.Value.Str() != "" {
		result.ChannelAvatar = avatar.Value.Str()
	}

	// Extract subscriber count
	if subs, err := page.Eval(`() => {
		// Try to find subscriber count element
		const subEl = document.querySelector('yt-formatted-string#owner-sub-count');
		if (subEl) {
			// Get the text content (like "271K subscribers")
			const text = subEl.textContent?.trim() || '';
			// Remove the word "subscribers" to get just the count
			return text.replace(/\s*subscribers?\s*/i, '').trim();
		}
		// Alternative selector
		const altSubEl = document.querySelector('#owner-sub-count, [id*="subscriber-count"]');
		if (altSubEl) {
			const text = altSubEl.textContent?.trim() || '';
			return text.replace(/\s*subscribers?\s*/i, '').trim();
		}
		return '';
	}`); err == nil && subs != nil && subs.Value.Str() != "" {
		result.SubscriberCount = subs.Value.Str()
	}
}

// normalizeYouTubeDate converts YouTube date format (e.g., "Jul 29, 2025") to ISO format ("2025-07-29")
func (c *Crawler) normalizeYouTubeDate(dateStr string) string {
	// Parse common YouTube date formats
	formats := []string{
		"Jan 2, 2006",
		"January 2, 2006",
		"Jan 02, 2006",
		"January 02, 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format("2006-01-02")
		}
	}

	// If parsing fails, return empty string to use original
	return ""
}

func isYouTubePage(page *rod.Page) bool {
	if page == nil {
		return false
	}
	value, err := page.Eval(`() => window.location.hostname.includes('youtube.com')`)
	if err != nil || value == nil {
		return false
	}
	return value.Value.Bool()
}

// parseYouTubeEpoch converts a YouTube published date string to epoch seconds.
func (c *Crawler) parseYouTubeEpoch(dateStr string) int64 {
	if dateStr == "" {
		return 0
	}

	if normalized := c.normalizeYouTubeDate(dateStr); normalized != "" {
		if t, err := time.Parse("2006-01-02", normalized); err == nil {
			return t.Unix()
		}
	}

	formats := []string{
		"2006-01-02",
		"Jan 2, 2006",
		"January 2, 2006",
		"Jan 02, 2006",
		"January 02, 2006",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Unix()
		}
	}

	return 0
}

func parseDurationSeconds(duration string) int {
	if duration == "" {
		return 0
	}

	parts := strings.Split(strings.TrimSpace(duration), ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0
	}

	total := 0
	for _, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return 0
		}
		total = total*60 + value
	}

	return total
}
