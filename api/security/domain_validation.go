package security

import (
	"fmt"
	"net"
	"strings"
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

// privateNetworks are CIDR ranges for private/reserved IPs.
var privateNetworks = []net.IPNet{
	parseCIDR("127.0.0.0/8"),     // Loopback
	parseCIDR("10.0.0.0/8"),      // RFC1918
	parseCIDR("172.16.0.0/12"),   // RFC1918
	parseCIDR("192.168.0.0/16"),  // RFC1918
	parseCIDR("169.254.0.0/16"), // Link-local / cloud metadata
	parseCIDR("0.0.0.0/8"),       // "This" network
	parseCIDR("::1/128"),         // IPv6 loopback
	parseCIDR("fc00::/7"),        // IPv6 ULA
	parseCIDR("fe80::/10"),       // IPv6 link-local
}

func parseCIDR(cidr string) net.IPNet {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		panic("invalid CIDR in blocklist: " + cidr)
	}
	return *n
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
	// Strip port if present
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
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

func isPrivateIP(ip net.IP) bool {
	for _, n := range privateNetworks {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
