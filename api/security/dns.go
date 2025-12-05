package security

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type dnsTask struct{}

func newDNSTask() ScanTask {
	return dnsTask{}
}

func (dnsTask) Name() string {
	return "dns"
}

func (dnsTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	analysis, err := scanner.checkDNS(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.DNSRecords = *analysis
	return nil
}

type dnsResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
	LookupMX(ctx context.Context, host string) ([]*net.MX, error)
	LookupTXT(ctx context.Context, host string) ([]string, error)
	LookupNS(ctx context.Context, host string) ([]*net.NS, error)
	LookupCNAME(ctx context.Context, host string) (string, error)
}

type netResolver struct{}

func (netResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	resolver := &net.Resolver{}
	return resolver.LookupIPAddr(ctx, host)
}

func (netResolver) LookupMX(ctx context.Context, host string) ([]*net.MX, error) {
	resolver := &net.Resolver{}
	return resolver.LookupMX(ctx, host)
}

func (netResolver) LookupTXT(ctx context.Context, host string) ([]string, error) {
	resolver := &net.Resolver{}
	return resolver.LookupTXT(ctx, host)
}

func (netResolver) LookupNS(ctx context.Context, host string) ([]*net.NS, error) {
	resolver := &net.Resolver{}
	return resolver.LookupNS(ctx, host)
}

func (netResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	resolver := &net.Resolver{}
	return resolver.LookupCNAME(ctx, host)
}

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

	resolver := netResolver{}

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
	s.evaluateCAA(ctx, host, analysis)

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

func (s *Scanner) evaluateDMARC(ctx context.Context, resolver dnsResolver, host string, analysis *DNSAnalysis) {
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

func (s *Scanner) evaluateCAA(ctx context.Context, host string, analysis *DNSAnalysis) {
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
	conn, err := net.DialTimeout("udp", "1.1.1.1:53", 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	id := uint16(time.Now().UnixNano())
	question, err := buildCAAQuery(host, id)
	if err != nil {
		return nil, err
	}

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	} else {
		conn.SetDeadline(time.Now().Add(3 * time.Second))
	}

	if _, err := conn.Write(question); err != nil {
		return nil, err
	}

	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return parseCAAResponse(buf[:n], id)
}

func buildCAAQuery(host string, id uint16) ([]byte, error) {
	host = dnsName(host)
	var buf []byte
	buf = append(buf, byte(id>>8), byte(id))
	buf = append(buf, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00)
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if label == "" {
			continue
		}
		if len(label) > 63 {
			return nil, fmt.Errorf("label %s too long", label)
		}
		buf = append(buf, byte(len(label)))
		buf = append(buf, label...)
	}
	buf = append(buf, 0)
	buf = append(buf, 0x01, 0x01, 0x00, 0x01)
	return buf, nil
}

func dnsName(host string) string {
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return ""
	}
	return host
}

func parseCAAResponse(resp []byte, id uint16) ([]caaRecord, error) {
	if len(resp) < 12 {
		return nil, fmt.Errorf("invalid DNS response")
	}
	if uint16(resp[0])<<8|uint16(resp[1]) != id {
		return nil, fmt.Errorf("mismatched DNS ID")
	}
	qdCount := int(resp[4])<<8 | int(resp[5])
	ansCount := int(resp[6])<<8 | int(resp[7])
	idx := 12
	for i := 0; i < qdCount; i++ {
		var err error
		idx, err = skipName(resp, idx)
		if err != nil {
			return nil, err
		}
		idx += 4
	}
	records := make([]caaRecord, 0, ansCount)
	for i := 0; i < ansCount && idx < len(resp); i++ {
		var err error
		idx, err = skipName(resp, idx)
		if err != nil {
			return nil, err
		}
		if idx+10 > len(resp) {
			return nil, fmt.Errorf("short DNS answer")
		}
		typeCode := uint16(resp[idx])<<8 | uint16(resp[idx+1])
		idx += 8
		rdlen := int(resp[idx])<<8 | int(resp[idx+1])
		idx += 2
		if idx+rdlen > len(resp) {
			return nil, fmt.Errorf("short RDATA")
		}
		if typeCode == 257 && rdlen >= 3 {
			flag := resp[idx]
			tagLen := int(resp[idx+1])
			if 2+tagLen > rdlen {
				return nil, fmt.Errorf("invalid CAA tag length")
			}
			tag := string(resp[idx+2 : idx+2+tagLen])
			value := string(resp[idx+2+tagLen : idx+rdlen])
			records = append(records, caaRecord{Flag: flag, Tag: tag, Value: value})
		}
		idx += rdlen
	}
	return records, nil
}

func skipName(msg []byte, idx int) (int, error) {
	for {
		if idx >= len(msg) {
			return 0, fmt.Errorf("short name")
		}
		length := int(msg[idx])
		idx++
		if length == 0 {
			return idx, nil
		}
		if length&0xC0 == 0xC0 {
			if idx >= len(msg) {
				return 0, fmt.Errorf("short pointer")
			}
			return idx + 1, nil
		}
		idx += length
	}
}
