// dns.go
package security

import (
	"net"
	"strings"
)

func (s *Scanner) checkDNS(domain string) (*DNSAnalysis, error) {
	host := strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")

	analysis := &DNSAnalysis{
		ARecords:     make([]string, 0),
		MXRecords:    make([]string, 0),
		TXTRecords:   make([]string, 0),
		NSRecords:    make([]string, 0),
		CNAMERecords: make([]string, 0),
		Findings:     make([]string, 0),
	}

	// A records
	ips, err := net.LookupIP(host)
	if err == nil {
		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				analysis.ARecords = append(analysis.ARecords, ipv4.String())
			}
		}
	}

	// MX records
	if mxRecords, err := net.LookupMX(host); err == nil {
		for _, mx := range mxRecords {
			analysis.MXRecords = append(analysis.MXRecords, mx.Host)
		}
	}

	// TXT records
	if txtRecords, err := net.LookupTXT(host); err == nil {
		analysis.TXTRecords = append(analysis.TXTRecords, txtRecords...)
	}

	// NS records
	if nsRecords, err := net.LookupNS(host); err == nil {
		for _, ns := range nsRecords {
			analysis.NSRecords = append(analysis.NSRecords, ns.Host)
		}
	}

	// CNAME records
	if cname, err := net.LookupCNAME(host); err == nil {
		analysis.CNAMERecords = append(analysis.CNAMERecords, cname)
	}

	return analysis, nil
}
