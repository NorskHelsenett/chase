// Package netguard keeps the screenshot service from being used as an SSRF
// proxy into loopback, link-local/cloud-metadata, and other private or
// reserved address space.
//
// 10.0.0.0/8 is intentionally NOT blocked: Chase monitors internal hosts on
// that range, and the screenshot service must be able to capture them.
package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// ErrBlocked is returned (wrapped) whenever a target is refused by policy.
var ErrBlocked = errors.New("target address is not allowed")

// blockedTLDs mirrors api/security/domain_validation.go.
var blockedTLDs = []string{
	".local",
	".internal",
	".localhost",
	".lan",
	".home",
	".corp",
	".intranet",
	".cluster.local",
	".svc.cluster.local",
}

// blockedHosts mirrors api/security/domain_validation.go.
var blockedHosts = []string{
	"localhost",
	"kubernetes",
	"kubernetes.default",
	"kubernetes.default.svc",
}

var blockedNets = mustParseCIDRs(
	"0.0.0.0/8",       // "this" network
	"127.0.0.0/8",     // loopback
	"169.254.0.0/16",  // link-local / cloud metadata
	"172.16.0.0/12",   // RFC1918
	"192.168.0.0/16",  // RFC1918
	"100.64.0.0/10",   // CGNAT
	"192.0.0.0/24",    // IETF protocol assignments
	"192.0.2.0/24",    // TEST-NET-1
	"198.18.0.0/15",   // benchmarking
	"198.51.100.0/24", // TEST-NET-2
	"203.0.113.0/24",  // TEST-NET-3
	"224.0.0.0/4",     // multicast
	"240.0.0.0/4",     // reserved (includes 255.255.255.255)
	"::/128",          // unspecified
	"::1/128",         // loopback
	"fe80::/10",       // link-local
	"fc00::/7",        // unique local
	"ff00::/8",        // multicast
	// 10.0.0.0/8 deliberately allowed — see package doc.
)

func mustParseCIDRs(cidrs ...string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic("netguard: invalid CIDR " + c)
		}
		out = append(out, n)
	}
	return out
}

// IsBlockedIP reports whether ip falls in a blocked range. IPv4-mapped IPv6
// addresses (::ffff:a.b.c.d) are normalised to IPv4 first.
func IsBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// IsBlockedHostname reports whether host is an internal name by policy.
func IsBlockedHostname(host string) bool {
	host = normalizeHost(host)
	if host == "" {
		return true
	}
	for _, h := range blockedHosts {
		if host == h {
			return true
		}
	}
	for _, suffix := range blockedTLDs {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	return strings.ToLower(strings.TrimSuffix(host, "."))
}

// Resolver is the subset of *net.Resolver used by CheckHost (overridable in tests).
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// DefaultResolver is used by CheckHost and CheckURL.
var DefaultResolver Resolver = net.DefaultResolver

// CheckHost rejects internal hostnames and hosts that resolve to any blocked IP.
// A DNS failure is not treated as blocked; the subsequent fetch fails on its own
// and the connect-time guard still applies.
func CheckHost(ctx context.Context, host string) error {
	host = normalizeHost(host)
	if IsBlockedHostname(host) {
		return fmt.Errorf("%w: internal hostname %q", ErrBlocked, host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if IsBlockedIP(ip) {
			return fmt.Errorf("%w: %s", ErrBlocked, ip)
		}
		return nil
	}
	addrs, err := DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil
	}
	for _, a := range addrs {
		if IsBlockedIP(a.IP) {
			return fmt.Errorf("%w: %q resolves to %s", ErrBlocked, host, a.IP)
		}
	}
	return nil
}

// CheckURL validates scheme (http/https only) and host of rawURL.
func CheckURL(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: invalid URL", ErrBlocked)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: scheme %q", ErrBlocked, u.Scheme)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("%w: missing host", ErrBlocked)
	}
	return CheckHost(ctx, u.Hostname())
}

// DialControl is a net.Dialer Control hook that refuses to connect to blocked
// IPs. It runs after DNS resolution on the exact address being dialed, so it
// also covers redirects and DNS rebinding.
func DialControl(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBlocked, err)
	}
	if IsBlockedIP(net.ParseIP(host)) {
		return fmt.Errorf("%w: %s", ErrBlocked, host)
	}
	return nil
}

// NewHTTPClient returns a client whose every connection (including redirect
// hops) is checked by DialControl. Environment proxies are ignored so the
// check cannot be bypassed by dialing a proxy instead of the target.
func NewHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: timeout, Control: DialControl}
	transport := &http.Transport{
		Proxy:               nil,
		DialContext:         dialer.DialContext,
		TLSHandshakeTimeout: timeout,
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("%w: redirect scheme %q", ErrBlocked, req.URL.Scheme)
			}
			if IsBlockedHostname(req.URL.Hostname()) {
				return fmt.Errorf("%w: redirect to internal hostname", ErrBlocked)
			}
			return nil
		},
	}
}
