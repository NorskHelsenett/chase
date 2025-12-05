// infrastructure.go
package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
)

func (s *Scanner) checkInfrastructure(ctx context.Context, domain string) (*InfrastructureAnalysis, error) {
	resp, err := s.fetch(ctx, domain, requestOptions{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	host := strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	ipv4 := ""
	ipv6 := make([]string, 0)
	for _, ip := range ips {
		if ip.To4() != nil {
			if ipv4 == "" {
				ipv4 = ip.String()
			}
		} else if ip.To16() != nil {
			ipv6 = append(ipv6, ip.String())
		}
	}
	if ipv4 == "" && len(ips) > 0 {
		ipv4 = ips[0].String()
	}

	analysis := &InfrastructureAnalysis{
		IPAddress:     ipv4,
		IPv6Addresses: ipv6,
		HTTPStatus:    resp.Status,
		HTTPVersion:   resp.Proto,
		Server:        resp.Header.Get("Server"),
		CDNProvider:   detectCDN(resp.Header.Get("Server"), resp.Header),
		Technology:    make([]TechnologyAnalysis, 0),
		Risk:          RiskLow,
		Findings:      make([]Finding, 0),
	}

	if len(ipv6) == 0 {
		analysis.Findings = append(analysis.Findings, Finding{
			Description: "IPv6 not configured",
			Risk:        RiskLow,
			Evidence:    "No AAAA records detected",
			Mitigation:  "Provide IPv6 connectivity for parity",
		})
	}

	if analysis.CDNProvider != "" {
		analysis.Findings = append(analysis.Findings, Finding{
			Description: "Site appears to be served via CDN",
			Risk:        RiskLow,
			Evidence:    analysis.CDNProvider,
			Mitigation:  "Ensure CDN security features (WAF, TLS) are enabled",
		})
	}

	if resp.ProtoMajor < 2 {
		analysis.Findings = append(analysis.Findings, Finding{
			Description: "HTTP/2 not negotiated",
			Risk:        RiskLow,
			Evidence:    resp.Proto,
			Mitigation:  "Enable HTTP/2 or HTTP/3 to improve resilience",
		})
	}

	// Check server information disclosure
	if analysis.Server != "" {
		analysis.Findings = append(analysis.Findings, Finding{
			Description: "Server header reveals technology information",
			Risk:        RiskMedium,
			Evidence:    analysis.Server,
			Mitigation:  "Remove or customize the Server header",
		})
		analysis.Risk = RiskMedium
	}

	// Detect common technologies from headers
	techHeaders := map[string]string{
		"X-Powered-By":     "Backend Technology",
		"X-AspNet-Version": "ASP.NET",
		"X-Runtime":        "Ruby",
	}

	for header, techName := range techHeaders {
		if value := resp.Header.Get(header); value != "" {
			analysis.Technology = append(analysis.Technology, TechnologyAnalysis{
				Name:    techName,
				Version: value,
			})
			analysis.Findings = append(analysis.Findings, Finding{
				Description: "Technology version disclosed",
				Risk:        RiskMedium,
				Evidence:    fmt.Sprintf("%s: %s", techName, value),
				Mitigation:  "Remove technology version headers",
			})
		}
	}

	return analysis, nil
}

func detectCDN(serverHeader string, headers http.Header) string {
	headerValues := []string{
		serverHeader,
		headers.Get("Via"),
		headers.Get("X-CDN"),
		headers.Get("CF-Ray"),
		headers.Get("X-Served-By"),
	}

	for _, value := range headerValues {
		value = strings.ToLower(value)
		switch {
		case strings.Contains(value, "cloudflare"):
			return "Cloudflare"
		case strings.Contains(value, "akamai"):
			return "Akamai"
		case strings.Contains(value, "cloudfront"):
			return "AWS CloudFront"
		case strings.Contains(value, "fastly"):
			return "Fastly"
		case strings.Contains(value, "azureedge"):
			return "Azure CDN"
		case strings.Contains(value, "cdn77"):
			return "CDN77"
		}
	}
	return ""
}

type infrastructureTask struct{}

func newInfrastructureTask() ScanTask {
	return infrastructureTask{}
}

func (infrastructureTask) Name() string {
	return "infrastructure"
}

func (infrastructureTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	infrastructure, err := scanner.checkInfrastructure(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.Infrastructure = *infrastructure
	return nil
}
