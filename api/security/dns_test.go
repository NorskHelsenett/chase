package security

import (
    "context"
    "net"
    "testing"
)

type fakeResolver struct {
    ip   []net.IPAddr
    mx   []*net.MX
    txt  map[string][]string
    ns   []*net.NS
    cname string
    err   error
}

func (f fakeResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
    return f.ip, f.err
}

func (f fakeResolver) LookupMX(ctx context.Context, host string) ([]*net.MX, error) {
    return f.mx, f.err
}

func (f fakeResolver) LookupTXT(ctx context.Context, host string) ([]string, error) {
    if f.txt == nil {
        return nil, f.err
    }
    return f.txt[host], f.err
}

func (f fakeResolver) LookupNS(ctx context.Context, host string) ([]*net.NS, error) {
    return f.ns, f.err
}

func (f fakeResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
    return f.cname, f.err
}

func TestEvaluateSPF(t *testing.T) {
    analysis := &DNSAnalysis{}
    analysis.TXTRecords = []string{"v=spf1 include:_spf.example.com ~all"}
    (&Scanner{}).evaluateSPF(analysis)
    if len(analysis.Findings) != 1 {
        t.Fatalf("expected finding for missing SPF strictness, got %#v", analysis.Findings)
    }
}
