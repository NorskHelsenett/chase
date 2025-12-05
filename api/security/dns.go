package security

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

func (s *Scanner) checkDNS(ctx context.Context, domain string) (*DNSAnalysis, error) {
	host := strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")

	analysis := &DNSAnalysis{
		ARecords:     make([]string, 0),
		MXRecords:    make([]string, 0),
		TXTRecords:   make([]string, 0),
		NSRecords:    make([]string, 0),
		CNAMERecords: make([]string, 0),
		Findings:     make([]string, 0),
	}

	resolver := &net.Resolver{}

	if ips, err := resolver.LookupIPAddr(ctx, host); err == nil {
		for _, ip := range ips {
			if ipv4 := ip.IP.To4(); ipv4 != nil {
				analysis.ARecords = append(analysis.ARecords, ipv4.String())
			}
		}
	}

	if mxRecords, err := resolver.LookupMX(ctx, host); err == nil {
		for _, mx := range mxRecords {
			analysis.MXRecords = append(analysis.MXRecords, mx.Host)
		}
	}

	if txtRecords, err := resolver.LookupTXT(ctx, host); err == nil {
		analysis.TXTRecords = append(analysis.TXTRecords, txtRecords...)
	}

	if nsRecords, err := resolver.LookupNS(ctx, host); err == nil {
		for _, ns := range nsRecords {
			analysis.NSRecords = append(analysis.NSRecords, ns.Host)
		}
	}

	if cname, err := resolver.LookupCNAME(ctx, host); err == nil {
		analysis.CNAMERecords = append(analysis.CNAMERecords, cname)
	}

	s.evaluateSPF(analysis)
	s.evaluateDMARC(ctx, resolver, host, analysis)
	s.evaluateCAA(ctx, resolver, host, analysis)

	return analysis, nil
}

func (s *Scanner) evaluateSPF(analysis *DNSAnalysis) {
	hasSPF := false
	for _, record := range analysis.TXTRecords {
		lower := strings.ToLower(record)
		if strings.HasPrefix(lower, "v=spf1") {
			hasSPF = true
			if strings.Contains(lower, "+all") {
				analysis.Findings = append(analysis.Findings,
					"SPF record allows all senders (+all) – tighten to specific senders only")
			}
		}
	}

	if !hasSPF {
		analysis.Findings = append(analysis.Findings, "Missing SPF record")
	}
}

func (s *Scanner) evaluateDMARC(ctx context.Context, resolver *net.Resolver, host string, analysis *DNSAnalysis) {
	records, err := resolver.LookupTXT(ctx, "_dmarc."+host)
	if err != nil || len(records) == 0 {
		analysis.Findings = append(analysis.Findings, "Missing DMARC record")
		return
	}

	analysis.TXTRecords = append(analysis.TXTRecords, records...)

	for _, record := range records {
		lower := strings.ToLower(record)
		if strings.Contains(lower, "p=none") {
			analysis.Findings = append(analysis.Findings, "DMARC policy set to none – enforce quarantine or reject")
		}
		if !strings.Contains(lower, "rua=") {
			analysis.Findings = append(analysis.Findings, "DMARC record missing aggregate reporting (rua)")
		}
	}
}

func (s *Scanner) evaluateCAA(ctx context.Context, resolver *net.Resolver, host string, analysis *DNSAnalysis) {
	records, err := lookupCAARecords(ctx, host)
	if err != nil {
		if errors.Is(err, errCAAUnsupported) {
			analysis.Findings = append(analysis.Findings,
				"CAA lookup unavailable in current runtime")
		} else {
			analysis.Findings = append(analysis.Findings,
				fmt.Sprintf("CAA lookup failed: %v", err))
		}
		return
	}

	if len(records) == 0 {
		analysis.Findings = append(analysis.Findings, "Missing CAA records – any CA can issue certificates")
		return
	}

	for _, record := range records {
		analysis.Findings = append(analysis.Findings,
			fmt.Sprintf("CAA policy: %s (tag=%s flag=%d)", record.Value, record.Tag, record.Flag))
	}
}

var errCAAUnsupported = errors.New("CAA lookup unsupported")

type caaRecord struct {
	Flag  uint8
	Tag   string
	Value string
}

func lookupCAARecords(ctx context.Context, host string) ([]caaRecord, error) {
	return nil, errCAAUnsupported
}
