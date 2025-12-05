package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type apiProbe struct {
	Path        string
	Description string
	Risk        RiskLevel
	Method      string
	Headers     http.Header
	Body        []byte
	Validator   func([]byte, *http.Response) bool
}

func (s *Scanner) checkAPIExposures(ctx context.Context, domain string) (*APIExposureAnalysis, error) {
	probes := []apiProbe{
		{
			Path:        "/graphql",
			Description: "GraphQL endpoint",
			Risk:        RiskHigh,
			Validator: func(body []byte, resp *http.Response) bool {
				content := strings.ToLower(string(body))
				return strings.Contains(content, "must provide query string") ||
					strings.Contains(content, "\"errors\"") ||
					strings.Contains(content, "graphql")
			},
		},
		{
			Path:        "/graphql",
			Method:      http.MethodPost,
			Description: "GraphQL introspection enabled",
			Risk:        RiskMedium,
			Headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: []byte(`{"query":"query IntrospectionQuery { __schema { queryType { name } } }"}`),
			Validator: func(body []byte, resp *http.Response) bool {
				if resp.StatusCode >= 400 {
					return false
				}
				return bytes.Contains(bytes.ToLower(body), []byte("__schema"))
			},
		},
		{
			Path:        "/graphiql",
			Description: "GraphiQL interface",
			Risk:        RiskHigh,
			Validator: func(body []byte, resp *http.Response) bool {
				content := strings.ToLower(string(body))
				return strings.Contains(content, "graphiql")
			},
		},
		{
			Path:        "/playground",
			Description: "GraphQL Playground",
			Risk:        RiskHigh,
			Validator: func(body []byte, resp *http.Response) bool {
				return strings.Contains(strings.ToLower(string(body)), "graphql playground")
			},
		},
		{
			Path:        "/metrics",
			Description: "Prometheus metrics",
			Risk:        RiskMedium,
			Validator: func(body []byte, resp *http.Response) bool {
				content := string(body)
				return strings.HasPrefix(content, "# HELP") || strings.Contains(content, "process_cpu_seconds_total")
			},
		},
		{
			Path:        "/actuator",
			Description: "Spring Boot actuator",
			Risk:        RiskMedium,
			Validator: func(body []byte, resp *http.Response) bool {
				content := strings.ToLower(string(body))
				return strings.Contains(content, "\"_links\"") && strings.Contains(content, "actuator")
			},
		},
		{
			Path:        "/console",
			Description: "Administrative console",
			Risk:        RiskHigh,
			Validator: func(body []byte, resp *http.Response) bool {
				content := strings.ToLower(string(body))
				return strings.Contains(content, "admin console") || strings.Contains(content, "management console")
			},
		},
		{
			Path:        "/.well-known/openid-configuration",
			Description: "OpenID Connect discovery document exposed",
			Risk:        RiskMedium,
			Validator: func(body []byte, resp *http.Response) bool {
				if resp.StatusCode >= 400 {
					return false
				}
				content := strings.ToLower(string(body))
				return strings.Contains(content, "token_endpoint") && strings.Contains(content, "authorization_endpoint")
			},
		},
	}

	analysis := &APIExposureAnalysis{
		Endpoints: make([]string, 0),
		Risk:      RiskLow,
		Findings:  make([]Finding, 0),
	}

	for _, probe := range probes {
		opts := requestOptions{}
		if probe.Method != "" {
			opts.method = probe.Method
		}
		if probe.Headers != nil {
			opts.headers = probe.Headers
		}
		if len(probe.Body) > 0 {
			opts.body = probe.Body
		}

		resp, err := s.fetch(ctx, domain+probe.Path, opts)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
		resp.Body.Close()
		if err != nil {
			continue
		}

		if resp.StatusCode >= http.StatusInternalServerError {
			continue
		}

		if probe.Validator(body, resp) {
			analysis.Endpoints = append(analysis.Endpoints, probe.Path)
			analysis.Findings = append(analysis.Findings, Finding{
				Description: probe.Description + " exposed at " + probe.Path,
				Risk:        probe.Risk,
				Evidence:    resp.Status,
				Mitigation:  "Restrict access or require authentication",
			})
			if probe.Risk > analysis.Risk {
				analysis.Risk = probe.Risk
			}
		}
	}

	s.probeLoginEndpoints(ctx, domain, analysis)

	return analysis, nil
}

func (s *Scanner) checkHealthProbes(ctx context.Context, domain string) (*HealthProbeAnalysis, error) {
	paths := []string{
		"/health",
		"/healthz",
		"/livez",
		"/readyz",
		"/liveness",
		"/readiness",
		"/actuator/health",
	}

	analysis := &HealthProbeAnalysis{
		Paths:    make(map[string]int),
		Risk:     RiskLow,
		Findings: make([]Finding, 0),
	}

	for _, path := range paths {
		resp, err := s.fetch(ctx, domain+path, requestOptions{})
		if err != nil {
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		resp.Body.Close()
		if err != nil {
			continue
		}

		analysis.Paths[path] = resp.StatusCode
		if resp.StatusCode >= 400 {
			if isLikelySoft404(body) {
				continue
			}
			continue
		}

		if exposesDetails(body) {
			analysis.Findings = append(analysis.Findings, Finding{
				Description: "Health endpoint " + path + " exposes internal details",
				Risk:        RiskMedium,
				Evidence:    truncateEvidence(body),
				Mitigation:  "Serve health probes on a separate authenticated channel",
			})
			if analysis.Risk < RiskMedium {
				analysis.Risk = RiskMedium
			}
		}
	}

	return analysis, nil
}

func exposesDetails(body []byte) bool {
	lower := strings.ToLower(string(body))
	if strings.Contains(lower, "\"status\"") && strings.Contains(lower, "\"details\"") {
		return true
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		if len(payload) > 1 {
			return true
		}
	}

	return strings.Contains(lower, "ok") && len(body) > 20
}

func truncateEvidence(body []byte) string {
	if len(body) > 200 {
		return string(body[:200]) + "..."
	}
	return string(body)
}

func isLikelySoft404(body []byte) bool {
	lower := strings.ToLower(string(body))
	if len(body) > 8192 {
		return true
	}

	keywords := []string{
		"404 not found",
		"page not found",
		"siden finnes ikke",
		"seite nicht gefunden",
		"pagina no encontrada",
		"pagina não encontrada",
		"pagina non trovata",
		"страница не найдена",
		"页面未找到",
		"خطأ 404",
	}

	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	return false
}

func (s *Scanner) probeLoginEndpoints(ctx context.Context, domain string, analysis *APIExposureAnalysis) {
	loginPaths := []string{
		"/login",
		"/signin",
		"/account/login",
		"/auth/login",
		"/admin/login",
		"/Account/Login",
	}

	for _, path := range loginPaths {
		resp, err := s.fetch(ctx, domain+path, requestOptions{})
		if err != nil {
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
		resp.Body.Close()
		if err != nil {
			continue
		}

		if resp.StatusCode >= http.StatusInternalServerError {
			continue
		}

		lower := strings.ToLower(string(body))
		if !strings.Contains(lower, "<form") || !strings.Contains(lower, "type=\"password\"") {
			continue
		}

		if !containsCSRFTokens(lower) {
			analysis.Findings = append(analysis.Findings, Finding{
				Description: "Login form missing CSRF token",
				Risk:        RiskHigh,
				Evidence:    path,
				Mitigation:  "Embed a hidden anti-CSRF token in the form and validate it server-side",
			})
			analysis.Endpoints = appendUniqueEndpoint(analysis.Endpoints, path)
			analysis.Risk = maxRiskLevel(analysis.Risk, RiskHigh)
		}

		if strings.HasPrefix(domain, "http://") {
			analysis.Findings = append(analysis.Findings, Finding{
				Description: "Login page served over HTTP",
				Risk:        RiskHigh,
				Evidence:    fmt.Sprintf("Password form exposed at %s%s", domain, path),
				Mitigation:  "Force HTTPS for authentication endpoints",
			})
			analysis.Endpoints = appendUniqueEndpoint(analysis.Endpoints, path)
			analysis.Risk = maxRiskLevel(analysis.Risk, RiskHigh)
		}
	}
}

func containsCSRFTokens(lower string) bool {
	indicators := []string{
		"csrf-token",
		"name=\"csrf\"",
		"name='csrf'",
		"name=\"_csrf\"",
		"name='__csrf'",
		"__requestverificationtoken",
		"anti-forgery",
		"_antiforgery",
	}

	for _, indicator := range indicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

func appendUniqueEndpoint(endpoints []string, path string) []string {
	for _, existing := range endpoints {
		if existing == path {
			return endpoints
		}
	}
	return append(endpoints, path)
}
