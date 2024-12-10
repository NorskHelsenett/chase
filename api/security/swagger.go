// swagger.go
package security

import (
	"fmt"
	"sync"
)

func (s *Scanner) checkSwaggerDocs(domain string) (*SwaggerAnalysis, error) {
	swaggerPaths := []string{
		"/swagger",
		"/swagger-ui.html",
		"/swagger/index.html",
		"/api-docs",
		"/openapi.json",
		"/swagger/v1/swagger.json",
		"/v1/swagger.json",
		"/v2/swagger.json",
		"/swagger-resources",
		"/swagger/ui",
	}

	analysis := &SwaggerAnalysis{
		Endpoints: make([]string, 0),
		Exposed:   false,
		Risk:      RiskLow,
		Findings:  make([]Finding, 0),
		Recommendations: []string{
			"Restrict access to API documentation",
			"Implement authentication for documentation access",
			"Consider hiding documentation in production",
			"Use API key authentication",
			"Implement rate limiting for API endpoints",
		},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, path := range swaggerPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			status, err := checkRealStatus(s.client, domain+p)
			if err != nil {
				return
			}

			if status != 404 {
				mu.Lock()
				analysis.Endpoints = append(analysis.Endpoints, p)
				analysis.Exposed = true
				analysis.Findings = append(analysis.Findings, Finding{
					Description: fmt.Sprintf("API documentation exposed at %s", p),
					Risk:        RiskHigh,
					Evidence:    fmt.Sprintf("Path %s returned status code %d", p, status),
					Mitigation:  "Restrict access to API documentation in production",
				})
				mu.Unlock()
			}
		}(path)
	}

	wg.Wait()

	if analysis.Exposed {
		analysis.Risk = RiskHigh
	}

	return analysis, nil
}
