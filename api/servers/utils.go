package servers

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func calculateNextCheckInterval(server Server) (time.Duration, bool) {
	// Look at recent history - last 7 days
	cutoff := time.Now().Add(-7 * 24 * time.Hour)

	var recentResults []PingResult
	for _, result := range server.PingResults {
		if result.Timestamp.After(cutoff) {
			recentResults = append(recentResults, result)
		}
	}

	if len(recentResults) == 0 {
		return time.Duration(server.UpdateInterval*3) * time.Minute, true // 3x base interval for new servers
	}

	// Calculate failure rate
	failureCount := 0
	for _, result := range recentResults {
		if result.StatusCode != server.ExpectedStatusCode || result.Error != "" {
			failureCount++
		}
	}
	failureRate := float64(failureCount) / float64(len(recentResults))

	// If we have enough data and very high failure rate, suggest deactivation
	if len(recentResults) >= 10 && failureRate > 0.95 {
		return 24 * time.Hour, false // Recommend deactivation
	}

	// Dynamic interval based on failure rate
	switch {
	case failureRate == 0:
		return time.Duration(float64(server.UpdateInterval)) * time.Minute, true
	case failureRate <= 0.1:
		return time.Duration(float64(server.UpdateInterval)*3) * time.Minute, true
	case failureRate <= 0.25:
		return time.Duration(float64(server.UpdateInterval)*6) * time.Minute, true
	case failureRate <= 0.5:
		return time.Duration(float64(server.UpdateInterval)*12) * time.Minute, true
	case failureRate <= 0.75:
		return time.Duration(float64(server.UpdateInterval)*36) * time.Minute, true
	default:
		return time.Duration(float64(server.UpdateInterval)*72) * time.Minute, true
	}
}

// @todo make the pingresults much smaller to save space
func pingServer(server Server) PingResult {
	result := PingResult{
		ServerID:  server.ID,
		Timestamp: time.Now(),
	}

	// Try HTTPS first, then fallback to HTTP if it fails
	schemes := []string{"https://", "http://"}
	var lastErr error

	for _, scheme := range schemes {
		fullURL := scheme + server.URL

		// Configure TLS
		tlsConfig := &tls.Config{
			InsecureSkipVerify: server.AllowInsecure,
		}

		// Create custom HTTP client with timeout
		transport := &http.Transport{
			TLSClientConfig: tlsConfig,
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
		}

		client := &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if !server.FollowRedirect {
					return http.ErrUseLastResponse
				}
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		}

		// Use ENV CHASE_HOSTNAME for User-Agent, else fallback to GitHub repo URL
		scannerURL := os.Getenv("CHASE_HOSTNAME")
		if scannerURL == "" {
			scannerURL = "https://github.com/NorskHelsenett/chase"
		}

		// Create request
		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("User-Agent", "ChaseMonitor/1.0 (+"+scannerURL+") Automated Security Scanner for "+server.URL)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

		startTime := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		// If we got here, the request was successful
		result.ResponseTime = float64(time.Since(startTime).Milliseconds())
		result.StatusCode = resp.StatusCode

		// Extract certificate and connection details if HTTPS
		if scheme == "https://" && resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
			result.PingDetail = extractConnectionDetails(resp, fullURL)
		}

		// Extract site metadata (favicon, title, etc.) if not yet stored
		if server.Favicon == "" && resp.StatusCode >= 200 && resp.StatusCode < 400 {
			ct := resp.Header.Get("Content-Type")
			if strings.Contains(ct, "text/html") {
				result.siteMetadata = extractSiteMetadata(resp.Body)
			}
		}

		return result
	}

	// If we get here, both HTTPS and HTTP failed
	result.Error = lastErr.Error()
	return result
}

// extractConnectionDetails extracts certificate and connection information from HTTP response
func extractConnectionDetails(resp *http.Response, url string) *PingDetail {
	detail := &PingDetail{
		RedirectCount: 0, // TODO: track redirects if needed
	}

	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]

		// Certificate validation
		detail.TLSValid = true
		detail.CertExpiryDate = cert.NotAfter
		detail.CertIssuer = cert.Issuer.CommonName
		detail.CertCommonName = cert.Subject.CommonName

		// Organization name
		if len(cert.Subject.Organization) > 0 {
			detail.OrganizationName = cert.Subject.Organization[0]
		}

		// Check if certificate is expired or expiring soon
		if time.Now().After(cert.NotAfter) {
			detail.TLSValid = false
		}
	}

	// Extract IP address from remote address
	if resp.Request != nil && resp.Request.RemoteAddr != "" {
		host, _, err := net.SplitHostPort(resp.Request.RemoteAddr)
		if err == nil {
			detail.IP = host
		}
	}

	return detail
}

// extractSiteMetadata reads the first 64KB of an HTML response to extract
// favicon, title, description, and og:image from the <head>.
func extractSiteMetadata(body io.Reader) SiteMetadata {
	limited := io.LimitReader(body, 64*1024)
	tokenizer := html.NewTokenizer(limited)

	var meta SiteMetadata
	var inTitle bool

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			if meta.Favicon == "" {
				meta.Favicon = "/favicon.ico"
			}
			return meta

		case html.TextToken:
			if inTitle && meta.Title == "" {
				meta.Title = strings.TrimSpace(string(tokenizer.Text()))
			}

		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			if string(tn) == "title" {
				inTitle = false
			}

		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := tokenizer.TagName()
			tag := string(tn)

			if tag == "body" {
				if meta.Favicon == "" {
					meta.Favicon = "/favicon.ico"
				}
				return meta
			}

			if tag == "title" {
				inTitle = true
				continue
			}

			if !hasAttr {
				continue
			}

			if tag == "link" {
				var rel, href string
				for {
					key, val, more := tokenizer.TagAttr()
					switch string(key) {
					case "rel":
						rel = strings.ToLower(string(val))
					case "href":
						href = string(val)
					}
					if !more {
						break
					}
				}
				if (rel == "icon" || rel == "shortcut icon") && href != "" && meta.Favicon == "" {
					meta.Favicon = href
				}
			}

			if tag == "meta" {
				var name, property, content string
				for {
					key, val, more := tokenizer.TagAttr()
					switch string(key) {
					case "name":
						name = strings.ToLower(string(val))
					case "property":
						property = strings.ToLower(string(val))
					case "content":
						content = string(val)
					}
					if !more {
						break
					}
				}
				if name == "description" && meta.Description == "" {
					meta.Description = content
				}
				if property == "og:image" && meta.OGImage == "" {
					meta.OGImage = content
				}
				if property == "og:title" && meta.Title == "" {
					meta.Title = content
				}
			}
		}
	}
}
