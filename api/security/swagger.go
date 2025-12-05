package security

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// SwaggerResponse represents a basic structure to validate Swagger/OpenAPI JSON
type SwaggerResponse struct {
	Swagger string                 `json:"swagger,omitempty"` // Swagger 2.0
	OpenAPI string                 `json:"openapi,omitempty"` // OpenAPI 3.0
	Info    map[string]interface{} `json:"info"`
	Paths   map[string]interface{} `json:"paths"`
}

func isSwaggerHTML(body string) bool {
	// Check for common Swagger UI HTML indicators
	indicators := []string{
		"swagger-ui",
		"SwaggerUIBundle",
		"swagger-ui.css",
		"swagger-ui-bundle.js",
	}

	for _, indicator := range indicators {
		if strings.Contains(strings.ToLower(body), strings.ToLower(indicator)) {
			return true
		}
	}
	return false
}

func validateSwaggerResponse(resp *http.Response) (bool, string) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

	// Check if it's HTML
	if strings.Contains(contentType, "text/html") {
		return isSwaggerHTML(string(body)), "HTML"
	}

	// Check if it's JSON
	if strings.Contains(contentType, "application/json") {
		var swaggerDoc SwaggerResponse
		if err := json.Unmarshal(body, &swaggerDoc); err != nil {
			return false, ""
		}

		// Validate basic Swagger/OpenAPI structure
		if (swaggerDoc.Swagger != "" || swaggerDoc.OpenAPI != "") &&
			swaggerDoc.Info != nil &&
			swaggerDoc.Paths != nil {
			return true, "JSON"
		}
	}

	return false, ""
}

func (s *Scanner) checkSwaggerDocs(ctx context.Context, domain string) (*SwaggerAnalysis, error) {
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

			resp, err := s.fetch(ctx, domain+p, requestOptions{})
			if err != nil {
				return
			}

			if resp.StatusCode == http.StatusOK {
				isValid, docType := validateSwaggerResponse(resp)
				if isValid {
					mu.Lock()
					analysis.Endpoints = append(analysis.Endpoints, p)
					analysis.Exposed = true
					analysis.Findings = append(analysis.Findings, Finding{
						Description: fmt.Sprintf("Valid Swagger %s documentation exposed at %s", docType, p),
						Risk:        RiskHigh,
						Evidence:    fmt.Sprintf("Path %s returned valid Swagger %s documentation", p, docType),
						Mitigation:  "Restrict access to API documentation in production",
					})
					mu.Unlock()
				}
			}
		}(path)
	}

	wg.Wait()

	if analysis.Exposed {
		analysis.Risk = RiskHigh
	}

	return analysis, nil
}
