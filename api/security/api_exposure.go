package security

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type apiProbe struct {
	Path        string
	Description string
	Risk        RiskLevel
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
	}

	analysis := &APIExposureAnalysis{
		Endpoints: make([]string, 0),
		Risk:      RiskLow,
		Findings:  make([]Finding, 0),
	}

	for _, probe := range probes {
		resp, err := s.fetch(ctx, domain+probe.Path, requestOptions{})
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
