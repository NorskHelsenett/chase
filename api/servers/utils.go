package servers

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/norskhelsenett/chase/security"
)

// calculateNextCheckInterval determines when to next check a server based on failure count
func calculateNextCheckInterval(failureCount int) time.Duration {
	switch {
	case failureCount == 0:
		return 15 * time.Minute
	case failureCount <= 6: // 1-6 failures: check every hour
		return 1 * time.Hour
	case failureCount <= 12: // 1-6 failures: check every 3 hours
		return 3 * time.Hour
	case failureCount <= 24: // 7-24 failures: check every 12 hours
		return 12 * time.Hour
	case failureCount <= 72: // 25-72 failures: check daily
		return 24 * time.Hour
	case failureCount <= 168: // 73-168 failures: check weekly
		return 7 * 24 * time.Hour
	default: // More than a week of failures: check bi-weekly
		return 14 * 24 * time.Hour
	}
}

func pingServer(server Server) PingResult {
	result := PingResult{
		ServerID:  server.ID,
		Timestamp: time.Now(),
	}

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
			result.RedirectCount = len(via)
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// Create request
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		result.Error = err.Error()
		server.FailureCount++
		return result
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		result.Error = err.Error()
		server.FailureCount++
		return result
	}
	defer resp.Body.Close()

	result.ResponseTime = float64(time.Since(startTime).Milliseconds())
	result.StatusCode = resp.StatusCode
	server.FailureCount = 0

	// Get IP address
	host := req.URL.Hostname()
	ips, err := net.LookupIP(host)
	if err == nil && len(ips) > 0 {
		result.IP = ips[0].String()
	}

	// Check TLS certificate
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		result.TLSValid = true
		result.CertExpiryDate = cert.NotAfter
		result.CertIssuer = cert.Issuer.CommonName
		result.CertCommonName = cert.Subject.CommonName
		result.OrganizationName = security.GetOrganization(cert)

		// Check if cert is expired or about to expire
		if time.Now().After(cert.NotAfter) {
			result.TLSValid = false
			result.Error = fmt.Sprintf("Certificate expired on %v", cert.NotAfter)
		}
	}

	return result
}
