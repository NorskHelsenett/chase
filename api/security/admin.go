// admin.go
package security

import (
	"fmt"
	"sync"
)

func (s *Scanner) checkAdminPages(domain string) (*AdminPagesAnalysis, error) {
	commonPaths := []string{
		"/admin",
		"/wp-admin",
		"/administrator",
		"/dashboard",
		"/panel",
		"/console",
		"/manage",
		"/backend",
		"/adm",
		"/control",
	}

	analysis := &AdminPagesAnalysis{
		Exposed:  make([]string, 0),
		Risk:     RiskLow,
		Findings: make([]Finding, 0),
		Recommendations: []string{
			"Implement IP-based access restrictions",
			"Use strong authentication mechanisms",
			"Enable two-factor authentication",
			"Change default admin URLs",
			"Use rate limiting on login attempts",
		},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, path := range commonPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			status, err := checkRealStatus(s.client, domain+p)
			if err != nil {
				return
			}

			if status != 404 {
				mu.Lock()
				analysis.Exposed = append(analysis.Exposed, p)
				analysis.Findings = append(analysis.Findings, Finding{
					Description: fmt.Sprintf("Admin interface potentially exposed at %s", p),
					Risk:        RiskHigh,
					Evidence:    fmt.Sprintf("Path %s returned status code %d", p, status),
					Mitigation:  "Restrict access and consider changing the URL",
				})
				mu.Unlock()
			}
		}(path)
	}

	wg.Wait()

	// Set risk level based on findings
	if len(analysis.Exposed) > 2 {
		analysis.Risk = RiskCritical
	} else if len(analysis.Exposed) > 0 {
		analysis.Risk = RiskHigh
	}

	return analysis, nil
}
