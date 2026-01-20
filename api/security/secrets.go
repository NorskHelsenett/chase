package security

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

const (
	maxSecretScanBytes   = 2 << 20
	maxSecretFindings    = 20
	maxSecretScriptFetch = 20
)

type secretPattern struct {
	name       string
	risk       RiskLevel
	regex      *regexp.Regexp
	mitigation string
}

var secretPatterns = []secretPattern{
	{
		name:       "OpenAI API Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "Hugging Face Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`hf_[A-Za-z0-9]{20,}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "NPM Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`npm_[A-Za-z0-9]{20,}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "Anthropic API Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`sk-ant-[A-Za-z0-9\-_]{20,}`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "Google API Key (Gemini/Firebase/GCP)",
		risk:       RiskMedium,
		regex:      regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),
		mitigation: "Restrict the key to approved origins and rotate it",
	},
	{
		name:       "AWS Access Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "AWS Temporary Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`ASIA[0-9A-Z]{16}`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "AWS Secret Access Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`(?i)aws_secret_access_key\s*[:=]\s*['\"][^'\"]{20,}['\"]`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "Stripe Secret Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24}`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "Stripe Restricted Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`rk_live_[0-9a-zA-Z]{24}`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "Slack Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`xox[baprs]-[0-9A-Za-z-]{10,48}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "GitHub Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`ghp_[A-Za-z0-9]{36}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "GitHub Fine-Grained Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`github_pat_[A-Za-z0-9_]{22,}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "GitLab Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`glpat-[A-Za-z0-9\-]{20,}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "Azure Storage Account Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`(?i)azure_storage_key\s*[:=]\s*['\"][^'\"]{20,}['\"]`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "Azure SAS Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`sv=\d{4}-\d{2}-\d{2}&ss=[a-z]+&srt=[a-z]+&sp=[rwdlacup]+&se=.+&sig=[A-Za-z0-9%]{20,}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "Azure DevOps Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`azdpat_[A-Za-z0-9]{20,}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "GCP Service Account Key",
		risk:       RiskCritical,
		regex:      regexp.MustCompile(`"type"\s*:\s*"service_account"`),
		mitigation: "Remove service account keys from client-side assets and rotate immediately",
	},
	{
		name:       "Twilio API Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`SK[0-9a-fA-F]{32}`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "Twilio Auth Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`(?i)twilio_auth_token\s*[:=]\s*['\"][^'\"]{20,}['\"]`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "SendGrid API Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`SG\.[A-Za-z0-9\-_]{20,}\.[A-Za-z0-9\-_]{20,}`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "Mailchimp API Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`(?i)[0-9a-f]{32}-us\d{1,2}`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "Mailgun API Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`key-[0-9a-fA-F]{32}`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "DigitalOcean Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`dop_v1_[a-f0-9]{64}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "Algolia API Key",
		risk:       RiskMedium,
		regex:      regexp.MustCompile(`(?i)algolia_api_key\s*[:=]\s*['\"][^'\"]{16,}['\"]`),
		mitigation: "Restrict the key to approved origins and rotate it",
	},
	{
		name:       "Mapbox Token",
		risk:       RiskMedium,
		regex:      regexp.MustCompile(`(pk|sk)\.[A-Za-z0-9_-]{60,}`),
		mitigation: "Restrict the token to approved origins and rotate it",
	},
	{
		name:       "Discord Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`(mfa\.[A-Za-z0-9_-]{80,}|[A-Za-z0-9_-]{24}\.[A-Za-z0-9_-]{6}\.[A-Za-z0-9_-]{27})`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "Telegram Bot Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`\d{9,10}:[A-Za-z0-9_-]{35}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "Sentry DSN",
		risk:       RiskMedium,
		regex:      regexp.MustCompile(`https?://[0-9a-f]{32}@[\w\.-]+/\d+`),
		mitigation: "Restrict DSN usage and rotate if exposed",
	},
	{
		name:       "Shopify Access Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`shpat_[a-f0-9]{32}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "Dropbox Access Token",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`sl\.[A-Za-z0-9_-]{20,}`),
		mitigation: "Remove credentials from client-side assets and rotate tokens",
	},
	{
		name:       "Supabase Service Key",
		risk:       RiskHigh,
		regex:      regexp.MustCompile(`(?i)supabase.*(service|secret).*key\s*[:=]\s*['\"][^'\"]{20,}['\"]`),
		mitigation: "Remove credentials from client-side assets and rotate keys",
	},
	{
		name:       "JWT Token",
		risk:       RiskMedium,
		regex:      regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`),
		mitigation: "Avoid embedding tokens in client assets; rotate exposed tokens",
	},
	{
		name:       "Private Key",
		risk:       RiskCritical,
		regex:      regexp.MustCompile(`-----BEGIN (RSA|DSA|EC|OPENSSH) PRIVATE KEY-----`),
		mitigation: "Remove private keys from public assets and rotate immediately",
	},
	{
		name:       "API Key Assignment",
		risk:       RiskMedium,
		regex:      regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*['\"][^'\"]{16,}['\"]`),
		mitigation: "Move secrets server-side and rotate any exposed keys",
	},
	{
		name:       "Token Assignment",
		risk:       RiskMedium,
		regex:      regexp.MustCompile(`(?i)token\s*[:=]\s*['\"][^'\"]{16,}['\"]`),
		mitigation: "Move secrets server-side and rotate any exposed tokens",
	},
}

type secretExposureTask struct{}

func newSecretExposureTask() ScanTask {
	return secretExposureTask{}
}

func (secretExposureTask) Name() string {
	return "secretExposure"
}

func (secretExposureTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	analysis, err := scanner.checkSecretExposure(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.SecretExposure = *analysis
	return nil
}

func (s *Scanner) checkSecretExposure(ctx context.Context, domain string) (*SecretExposureAnalysis, error) {
	analysis := &SecretExposureAnalysis{
		Findings: make([]Finding, 0),
		Risk:     RiskLow,
		Sources:  make([]string, 0),
		Checks:   make([]SecretCheck, 0),
	}

	checkState := defaultSecretChecks()

	htmlBody, contentType, err := s.fetchServiceContent(ctx, domain, "/.html", maxSecretScanBytes)
	if err != nil {
		analysis.Checks = buildSecretChecks(checkState)
		return analysis, err
	}
	if len(htmlBody) == 0 || (!strings.Contains(strings.ToLower(contentType), "html") && !isLikelyHTML(htmlBody)) {
		analysis.Checks = buildSecretChecks(checkState)
		return analysis, nil
	}

	seen := make(map[string]struct{})
	scanSecretContent(string(htmlBody), "document", analysis, seen, checkState)

	inlineScripts := extractInlineScripts(string(htmlBody))
	for idx, script := range inlineScripts {
		source := fmt.Sprintf("inline-script-%d", idx+1)
		scanSecretContent(script, source, analysis, seen, checkState)
		if len(analysis.Findings) >= maxSecretFindings {
			analysis.Checks = buildSecretChecks(checkState)
			return analysis, nil
		}
	}

	scriptSources := extractScriptSources(string(htmlBody))
	fetched := 0
	for _, src := range scriptSources {
		if fetched >= maxSecretScriptFetch || len(analysis.Findings) >= maxSecretFindings {
			break
		}
		absolute, err := resolveScriptURL(domain, src)
		if err != nil {
			continue
		}
		body, scriptType, err := s.fetchServiceContent(ctx, absolute, "/.html", maxSecretScanBytes)
		if err != nil || len(body) == 0 {
			continue
		}
		if strings.Contains(strings.ToLower(scriptType), "html") || isLikelyHTML(body) {
			continue
		}
		source := absolute
		scanSecretContent(string(body), source, analysis, seen, checkState)
		fetched++
	}

	analysis.Checks = buildSecretChecks(checkState)
	return analysis, nil
}

func (s *Scanner) fetchServiceContent(ctx context.Context, targetURL, suffix string, limit int64) ([]byte, string, error) {
	baseURL := os.Getenv("SCREENSHOT_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://screenshot:11235"
	}

	requestURL := buildServiceURL(baseURL, targetURL, suffix)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, "", err
	}

	client := &http.Client{
		Timeout:   serviceTimeout(),
		Transport: s.strictTransport,
	}
	resp, err := s.doRequest(client, req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("service returned status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, "", err
	}

	return content, resp.Header.Get("Content-Type"), nil
}

func buildServiceURL(baseURL, targetURL, suffix string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	targetURL = strings.TrimRight(targetURL, "/")
	suffix = "/" + strings.TrimPrefix(suffix, "/")
	return fmt.Sprintf("%s/%s%s", baseURL, targetURL, suffix)
}

func extractScriptSources(htmlBody string) []string {
	re := regexp.MustCompile(`(?is)<script[^>]*\ssrc=["']([^"']+)["'][^>]*>`)
	matches := re.FindAllStringSubmatch(htmlBody, -1)
	sources := make([]string, 0, len(matches))
	seen := make(map[string]struct{})
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		src := strings.TrimSpace(match[1])
		if src == "" {
			continue
		}
		if _, exists := seen[src]; exists {
			continue
		}
		seen[src] = struct{}{}
		sources = append(sources, src)
	}
	return sources
}

func extractInlineScripts(htmlBody string) []string {
	re := regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`)
	matches := re.FindAllStringSubmatch(htmlBody, -1)
	scripts := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		attrs := strings.ToLower(match[1])
		if strings.Contains(attrs, "src=") {
			continue
		}
		content := strings.TrimSpace(match[2])
		if content == "" {
			continue
		}
		scripts = append(scripts, content)
	}
	return scripts
}

func resolveScriptURL(baseURL, src string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(strings.TrimSpace(src))
	if err != nil {
		return "", err
	}
	return base.ResolveReference(ref).String(), nil
}

func scanSecretContent(content, source string, analysis *SecretExposureAnalysis, seen map[string]struct{}, checkState map[string]bool) {
	if content == "" {
		return
	}

	if _, exists := seen[source]; !exists {
		analysis.Sources = append(analysis.Sources, source)
		seen[source] = struct{}{}
	}

	for _, pattern := range secretPatterns {
		matches := pattern.regex.FindAllString(content, -1)
		for _, match := range matches {
			if len(analysis.Findings) >= maxSecretFindings {
				return
			}
			redacted := redactSecretMatch(match)
			key := pattern.name + ":" + redacted + ":" + source
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			analysis.Findings = append(analysis.Findings, Finding{
				Description: fmt.Sprintf("%s exposed in client assets", pattern.name),
				Risk:        pattern.risk,
				Evidence:    fmt.Sprintf("%s in %s", redacted, source),
				Mitigation:  pattern.mitigation,
			})
			analysis.Risk = maxRiskLevel(analysis.Risk, pattern.risk)
			if _, ok := checkState[pattern.name]; ok {
				checkState[pattern.name] = false
			}
		}
	}
}

func redactSecretMatch(match string) string {
	return strings.TrimSpace(match)
}

func defaultSecretChecks() map[string]bool {
	state := make(map[string]bool, len(secretPatterns))
	for _, pattern := range secretPatterns {
		state[pattern.name] = true
	}
	return state
}

func buildSecretChecks(state map[string]bool) []SecretCheck {
	checks := make([]SecretCheck, 0, len(secretPatterns))
	for _, pattern := range secretPatterns {
		passed, ok := state[pattern.name]
		if !ok {
			passed = true
		}
		checks = append(checks, SecretCheck{
			Name:   pattern.name,
			Passed: passed,
		})
	}
	return checks
}
