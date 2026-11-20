package netguard

import (
	"net"
	"testing"
)

func TestPolicies(t *testing.T) {
	cases := []struct {
		ip              string
		strict, monitor bool // expected blocked
	}{
		{"8.8.8.8", false, false},
		{"10.1.2.3", false, false},
		{"172.16.0.1", true, false},
		{"192.168.1.1", true, false},
		{"127.0.0.1", true, true},
		{"169.254.169.254", true, true},
		{"0.0.0.0", true, true},
		{"100.64.0.1", true, true},
		{"224.0.0.1", true, true},
		{"255.255.255.255", true, true},
		{"::1", true, true},
		{"::", true, true},
		{"fe80::1", true, true},
		{"fd00::1", true, false},
		{"::ffff:127.0.0.1", true, true},
		{"::ffff:169.254.169.254", true, true},
		{"2606:4700::1111", false, false},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if got := Strict.IsBlocked(ip); got != c.strict {
			t.Errorf("Strict.IsBlocked(%s) = %v, want %v", c.ip, got, c.strict)
		}
		if got := Monitor.IsBlocked(ip); got != c.monitor {
			t.Errorf("Monitor.IsBlocked(%s) = %v, want %v", c.ip, got, c.monitor)
		}
	}
}

func TestControl(t *testing.T) {
	if err := Monitor.Control("tcp", "127.0.0.1:80", nil); err == nil {
		t.Error("expected loopback dial to be rejected")
	}
	if err := Monitor.Control("tcp", "[::1]:443", nil); err == nil {
		t.Error("expected IPv6 loopback dial to be rejected")
	}
	if err := Monitor.Control("tcp", "10.0.0.5:443", nil); err != nil {
		t.Errorf("expected 10/8 dial to be allowed, got %v", err)
	}
}
