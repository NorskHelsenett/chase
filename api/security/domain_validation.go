package security

import (
	"fmt"
	"net"
	"strings"

	"github.com/norskhelsenett/chase/netguard"
)

// blockedTLDs are top-level domains or suffixes that indicate internal/non-routable services.
var blockedTLDs = []string{
	".local",
	".internal",
	".localhost",
	".lan",
	".home",
	".corp",
	".intranet",
	".cluster.local", // Kubernetes
	".svc.cluster.local",
}

// blockedHosts are exact hostnames that should never be scanned.
var blockedHosts = []string{
	"localhost",
	"kubernetes",
	"kubernetes.default",
	"kubernetes.default.svc",
}

// ValidateDomain checks whether a domain is safe to scan.
// Returns an error describing the reason if the domain is blocked.
func ValidateDomain(domain string) error {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fmt.Errorf("domain parameter is required")
	}

	// Strip protocol if present
	host := strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
	// Strip path, query and fragment (stored server URLs may include a path)
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	// Strip userinfo
	if i := strings.LastIndex(host, "@"); i >= 0 {
		host = host[i+1:]
	}
	// Strip port if present
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	host = strings.ToLower(strings.TrimSuffix(host, "."))

	// Check exact blocked hostnames
	for _, blocked := range blockedHosts {
		if host == blocked {
			return fmt.Errorf("scanning internal hostname %q is not allowed", host)
		}
	}

	// Check blocked TLD suffixes
	for _, suffix := range blockedTLDs {
		if strings.HasSuffix(host, suffix) {
			return fmt.Errorf("scanning internal domain %q (suffix %s) is not allowed", host, suffix)
		}
	}

	// If it's a raw IP address, check private ranges directly
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("scanning private/reserved IP %q is not allowed", host)
		}
	}

	// Resolve the domain and check all IPs
	ips, err := net.LookupIP(host)
	if err != nil {
		// DNS resolution failed — allow the scan to proceed and fail naturally
		return nil
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("domain %q resolves to private/reserved IP %s — scanning is not allowed", host, ip)
		}
	}

	return nil
}

// isPrivateIP reports whether ip is blocked for scans. The policy lives in
// netguard.Strict; 10.0.0.0/8 is intentionally allowed (internal monitored
// hosts), while loopback, metadata, reserved and other private ranges are not.
func isPrivateIP(ip net.IP) bool {
	return netguard.Strict.IsBlocked(ip)
}
