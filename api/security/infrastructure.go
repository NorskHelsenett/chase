// infrastructure.go
package security

import (
	"fmt"
	"net"
	"strings"
)

func (s *Scanner) checkInfrastructure(domain string) (*InfrastructureAnalysis, error) {
	resp, err := s.client.Get(domain)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	host := strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	analysis := &InfrastructureAnalysis{
		IPAddress:  ips[0].String(),
		HTTPStatus: resp.Status,
		Server:     resp.Header.Get("Server"),
		Technology: make([]TechnologyAnalysis, 0),
		Risk:       RiskLow,
		Findings:   make([]Finding, 0),
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
