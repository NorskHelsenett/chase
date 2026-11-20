package netguard

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true},
		{"127.255.255.254", true},
		{"0.0.0.0", true},
		{"169.254.169.254", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.1.1", true},
		{"100.64.0.1", true},
		{"192.0.0.1", true},
		{"192.0.2.10", true},
		{"198.18.0.1", true},
		{"198.51.100.7", true},
		{"203.0.113.9", true},
		{"224.0.0.1", true},
		{"255.255.255.255", true},
		{"::", true},
		{"::1", true},
		{"fe80::1", true},
		{"fd00::1", true},
		{"ff02::1", true},
		{"::ffff:127.0.0.1", true},
		{"::ffff:169.254.169.254", true},
		// Allowed
		{"10.0.0.5", false}, // internal monitored hosts — intentionally allowed
		{"10.255.255.255", false},
		{"::ffff:10.1.2.3", false},
		{"172.15.255.255", false},
		{"172.32.0.1", false},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"2001:4860:4860::8888", false},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("bad test IP %q", tt.ip)
			}
			if got := IsBlockedIP(ip); got != tt.blocked {
				t.Errorf("IsBlockedIP(%s) = %v, want %v", tt.ip, got, tt.blocked)
			}
		})
	}
	if !IsBlockedIP(nil) {
		t.Error("nil IP must be blocked")
	}
}

func TestIsBlockedHostname(t *testing.T) {
	tests := []struct {
		host    string
		blocked bool
	}{
		{"localhost", true},
		{"LOCALHOST.", true},
		{"kubernetes.default.svc", true},
		{"api.svc.cluster.local", true},
		{"metadata.google.internal", true},
		{"printer.local", true},
		{"", true},
		{"example.com", false},
		{"nhn.no", false},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := IsBlockedHostname(tt.host); got != tt.blocked {
				t.Errorf("IsBlockedHostname(%q) = %v, want %v", tt.host, got, tt.blocked)
			}
		})
	}
}

type fakeResolver map[string][]string

func (f fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := f[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	out := make([]net.IPAddr, 0, len(ips))
	for _, s := range ips {
		out = append(out, net.IPAddr{IP: net.ParseIP(s)})
	}
	return out, nil
}

func TestCheckURL(t *testing.T) {
	orig := DefaultResolver
	DefaultResolver = fakeResolver{
		"public.example":   {"93.184.216.34"},
		"internal.example": {"10.1.2.3"},
		"rebind.example":   {"93.184.216.34", "169.254.169.254"},
	}
	defer func() { DefaultResolver = orig }()

	tests := []struct {
		url     string
		blocked bool
	}{
		{"https://public.example/", false},
		{"https://internal.example/", false},
		{"http://10.0.0.5:8080/x", false},
		{"https://unresolvable.example/", false},
		{"https://rebind.example/", true},
		{"http://169.254.169.254/latest/meta-data/", true},
		{"http://127.0.0.1:6379/", true},
		{"http://[::1]/", true},
		{"http://[::ffff:127.0.0.1]/", true},
		{"http://localhost:8080/", true},
		{"http://host.containers.internal:8080/", true},
		{"file:///etc/passwd", true},
		{"ftp://public.example/", true},
		{"https:///nohost", true},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			err := CheckURL(context.Background(), tt.url)
			if tt.blocked {
				if !errors.Is(err, ErrBlocked) {
					t.Errorf("CheckURL(%q) = %v, want ErrBlocked", tt.url, err)
				}
			} else if err != nil {
				t.Errorf("CheckURL(%q) = %v, want nil", tt.url, err)
			}
		})
	}
}

func TestNewHTTPClientRefusesLoopbackAtDialTime(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewHTTPClient(2 * time.Second)
	_, err := client.Get(srv.URL) // httptest listens on 127.0.0.1
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected dial to loopback to be blocked, got %v", err)
	}
}
