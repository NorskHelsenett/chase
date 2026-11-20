// Package netguard blocks outbound connections from Chase to addresses that
// must never be reachable through user-influenced URLs (SSRF protection).
//
// Two policies exist:
//
//   - Strict is used for security scans and screenshots of arbitrary domains.
//     It blocks loopback, link-local/cloud-metadata, reserved ranges and the
//     private ranges 172.16.0.0/12, 192.168.0.0/16 and fc00::/7.
//     10.0.0.0/8 is deliberately ALLOWED because Chase monitors internal hosts
//     on that range; those endpoints are only reachable for logged-in users.
//   - Monitor is used for server pings and favicon/manifest fetches. Monitored
//     servers may live on any RFC1918 range, so only loopback, link-local,
//     metadata and reserved ranges are blocked.
//
// The check is enforced at connect time via net.Dialer.Control, so it also
// covers redirects and DNS rebinding (the IP checked is the IP dialed).
package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// Policy is a set of blocked networks.
type Policy struct {
	name    string
	blocked []*net.IPNet
}

// ErrBlocked is returned (wrapped) when a connection target is not allowed.
var ErrBlocked = errors.New("destination address is not allowed")

func mustCIDRs(cidrs ...string) []*net.IPNet {
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

// alwaysBlocked are never legitimate targets for outbound fetches.
var alwaysBlocked = []string{
	"0.0.0.0/8",         // "this" network
	"127.0.0.0/8",       // loopback
	"169.254.0.0/16",    // link-local / cloud metadata
	"100.64.0.0/10",     // CGNAT
	"192.0.0.0/24",      // IETF protocol assignments
	"192.0.2.0/24",      // TEST-NET-1
	"198.18.0.0/15",     // benchmarking
	"198.51.100.0/24",   // TEST-NET-2
	"203.0.113.0/24",    // TEST-NET-3
	"224.0.0.0/4",       // multicast
	"240.0.0.0/4",       // reserved + broadcast
	"::/128",            // unspecified
	"::1/128",           // IPv6 loopback
	"fe80::/10",         // IPv6 link-local
	"ff00::/8",          // IPv6 multicast
	"64:ff9b::/96",      // NAT64 (could map to internal IPv4)
	"2001:db8::/32",     // documentation
	"fd00:ec2::254/128", // AWS IMDS IPv6
}

var (
	// Strict blocks everything in alwaysBlocked plus private ranges except 10/8.
	Strict = &Policy{name: "strict", blocked: mustCIDRs(append(append([]string{}, alwaysBlocked...),
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7",
	)...)}
	// Monitor blocks only alwaysBlocked; RFC1918 targets are allowed.
	Monitor = &Policy{name: "monitor", blocked: mustCIDRs(alwaysBlocked...)}
)

// IsBlocked reports whether ip is not an allowed destination under p.
func (p *Policy) IsBlocked(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsUnspecified() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	for _, n := range p.blocked {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// CheckHost resolves host (a hostname or IP literal, without port) and returns
// an error if any resolved address is blocked. A DNS failure is returned as-is.
func (p *Policy) CheckHost(ctx context.Context, host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if p.IsBlocked(ip) {
			return fmt.Errorf("%w: %s", ErrBlocked, ip)
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return err
	}
	for _, a := range ips {
		if p.IsBlocked(a.IP) {
			return fmt.Errorf("%w: %s resolves to %s", ErrBlocked, host, a.IP)
		}
	}
	return nil
}

// Control is a net.Dialer Control func that rejects blocked destinations.
func (p *Policy) Control(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrBlocked, address)
	}
	if p.IsBlocked(net.ParseIP(host)) {
		return fmt.Errorf("%w: %s", ErrBlocked, host)
	}
	return nil
}

// Dialer returns a net.Dialer enforcing p.
func (p *Policy) Dialer(timeout time.Duration) *net.Dialer {
	return &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second, Control: p.Control}
}

// CheckRedirect returns an http.Client CheckRedirect func that enforces p on
// each redirect hop by hostname (useful when a proxy hides the dialed IP) and
// caps redirects at maxHops. If follow is false, the first response is returned.
func (p *Policy) CheckRedirect(follow bool, maxHops int) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if !follow {
			return http.ErrUseLastResponse
		}
		if len(via) >= maxHops {
			return fmt.Errorf("stopped after %d redirects", len(via))
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return fmt.Errorf("%w: redirect to scheme %q", ErrBlocked, req.URL.Scheme)
		}
		return p.CheckHost(req.Context(), req.URL.Hostname())
	}
}
