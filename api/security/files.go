package security

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Configuration thresholds for false positive detection
const (
	MaxExposedFiles  = 5       // Maximum reasonable number of exposed files
	SuspiciousRatio  = 0.8     // Threshold for suspicious detection rate
	MaxContentLength = 1 << 20 // 1MB max content size to check
)

// FileValidator defines validation logic for specific file types
type FileValidator func(content []byte) bool

// FileSignature contains metadata and validation logic for a file
type FileSignature struct {
	path        string
	fileType    string
	description string
	risk        RiskLevel
	validate    FileValidator
}

func (s *Scanner) checkFileExposure(ctx context.Context, domain string) (*FileExposureAnalysis, error) {
	analysis := &FileExposureAnalysis{
		ExposedFiles: make([]ExposedFile, 0),
		Risk:         RiskLow,
		Evidence:     make(map[string]string),
		Checks:       make([]FileCheck, 0),
	}

	commonFiles := []FileSignature{
		{
			path:        "/robots.txt",
			fileType:    "Config",
			description: "Robots exclusion file",
			risk:        RiskLow,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("User-agent:")) ||
					bytes.Contains(content, []byte("Disallow:"))
			},
		},
		{
			path:        "/sitemap.xml",
			fileType:    "Config",
			description: "Site map file",
			risk:        RiskLow,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("<?xml")) &&
					bytes.Contains(content, []byte("<urlset"))
			},
		},
		{
			path:        "/.git/HEAD",
			fileType:    "VCS",
			description: "Git repository exposure",
			risk:        RiskHigh,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("ref: refs/")) ||
					len(content) == 40 // Git SHA-1 hash length
			},
		},
		{
			path:        "/.env",
			fileType:    "Config",
			description: "Environment file exposure",
			risk:        RiskCritical,
			validate: func(content []byte) bool {
				return isLikelyEnvFile(content)
			},
		},
		{
			path:        "/.git/config",
			fileType:    "VCS",
			description: "Git config disclosure",
			risk:        RiskCritical,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("[core]")) && bytes.Contains(content, []byte("[remote"))
			},
		},
		{
			path:        "/.gitignore",
			fileType:    "Config",
			description: "Git ignore rules exposed",
			risk:        RiskMedium,
			validate: func(content []byte) bool {
				return len(content) > 0
			},
		},
		{
			path:        "/.svn/entries",
			fileType:    "VCS",
			description: "Subversion metadata exposed",
			risk:        RiskHigh,
			validate: func(content []byte) bool {
				return isLikelySubversionEntries(content)
			},
		},
		{
			path:        "/.hg/store/00manifest.i",
			fileType:    "VCS",
			description: "Mercurial repository exposed",
			risk:        RiskHigh,
			validate: func(content []byte) bool {
				return isLikelyMercurialRevlog(content)
			},
		},
		{
			path:        "/.DS_Store",
			fileType:    "Metadata",
			description: "macOS Finder metadata exposed",
			risk:        RiskLow,
			validate: func(content []byte) bool {
				return bytes.HasPrefix(content, []byte("Bud1"))
			},
		},
		{
			path:        "/.bash_history",
			fileType:    "Secrets",
			description: "Shell history exposed",
			risk:        RiskCritical,
			validate: func(content []byte) bool {
				return isLikelyShellHistory(content)
			},
		},
		{
			path:        "/.ssh/id_rsa",
			fileType:    "Secrets",
			description: "Private SSH key exposed",
			risk:        RiskCritical,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("BEGIN OPENSSH PRIVATE KEY")) ||
					bytes.Contains(content, []byte("BEGIN RSA PRIVATE KEY"))
			},
		},
		{
			path:        "/config.php",
			fileType:    "Config",
			description: "PHP configuration file exposed",
			risk:        RiskHigh,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("<?php"))
			},
		},
		{
			path:        "/web.config",
			fileType:    "Config",
			description: "IIS web.config exposed",
			risk:        RiskHigh,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("<configuration"))
			},
		},
		{
			path:        "/backup.zip",
			fileType:    "Backup",
			description: "Backup archive exposed",
			risk:        RiskCritical,
			validate: func(content []byte) bool {
				return len(content) > 4 && bytes.Equal(content[:4], []byte("PK\x03\x04"))
			},
		},
		{
			path:        "/database.sql",
			fileType:    "Backup",
			description: "Database dump exposed",
			risk:        RiskCritical,
			validate: func(content []byte) bool {
				return bytes.Contains(bytes.ToLower(content), []byte("insert into"))
			},
		},
		{
			path:        "/package-lock.json",
			fileType:    "Config",
			description: "Node lockfile exposed",
			risk:        RiskMedium,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("\"lockfileVersion\""))
			},
		},
		{
			path:        "/yarn.lock",
			fileType:    "Config",
			description: "Yarn lockfile exposed",
			risk:        RiskMedium,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("yarn lockfile v"))
			},
		},
		{
			path:        "/npm-shrinkwrap.json",
			fileType:    "Config",
			description: "npm shrinkwrap exposed",
			risk:        RiskMedium,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("\"dependencies\""))
			},
		},
		{
			path:        "/composer.json",
			fileType:    "Config",
			description: "Composer manifest exposed",
			risk:        RiskLow,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("\"require\""))
			},
		},
		{
			path:        "/.docker/config.json",
			fileType:    "Secrets",
			description: "Docker credentials exposed",
			risk:        RiskCritical,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("\"auths\""))
			},
		},
		{
			path:        "/node_modules/",
			fileType:    "Directory",
			description: "node_modules directory listing exposed",
			risk:        RiskMedium,
			validate: func(content []byte) bool {
				contentLower := strings.ToLower(string(content))
				return strings.Contains(contentLower, "index of /node_modules") ||
					strings.Contains(contentLower, "<title>index of")
			},
		},
		{
			path:        "/vendor/",
			fileType:    "Directory",
			description: "PHP vendor directory exposed",
			risk:        RiskMedium,
			validate: func(content []byte) bool {
				contentLower := strings.ToLower(string(content))
				return strings.Contains(contentLower, "index of /vendor") ||
					strings.Contains(contentLower, "<title>index of")
			},
		},
		{
			path:        "/.aws/credentials",
			fileType:    "Secrets",
			description: "AWS credentials file exposed",
			risk:        RiskCritical,
			validate: func(content []byte) bool {
				contentLower := strings.ToLower(string(content))
				return strings.Contains(contentLower, "[default]") && strings.Contains(contentLower, "aws_access_key_id")
			},
		},
		{
			path:        "/config/database.yml",
			fileType:    "Config",
			description: "Rails database configuration exposed",
			risk:        RiskHigh,
			validate: func(content []byte) bool {
				contentLower := strings.ToLower(string(content))
				return strings.Contains(contentLower, "adapter:") && strings.Contains(contentLower, "database:")
			},
		},
		{
			path:        "/appsettings.json",
			fileType:    "Config",
			description: ".NET appsettings exposed",
			risk:        RiskHigh,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("\"Logging\"")) && bytes.Contains(content, []byte("\"ConnectionStrings\""))
			},
		},
		{
			path:        "/storage/logs/laravel.log",
			fileType:    "Logs",
			description: "Laravel application logs exposed",
			risk:        RiskMedium,
			validate: func(content []byte) bool {
				contentLower := strings.ToLower(string(content))
				return strings.Contains(contentLower, "laravel") && strings.Contains(contentLower, "production")
			},
		},
		{
			path:        "/wp-config.php",
			fileType:    "Config",
			description: "WordPress configuration exposed",
			risk:        RiskCritical,
			validate: func(content []byte) bool {
				contentLower := strings.ToLower(string(content))
				return strings.Contains(contentLower, "define('db_") || strings.Contains(contentLower, "table_prefix")
			},
		},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	highestRisk := RiskLow
	totalChecked := 0
	checkState := make(map[string]bool, len(commonFiles))
	for _, file := range commonFiles {
		checkState[file.path] = true
	}

	for _, file := range commonFiles {
		wg.Add(1)
		go func(f FileSignature) {
			defer wg.Done()

			exposed, evidence := s.validateFile(ctx, domain, f)
			if exposed {
				mu.Lock()
				analysis.ExposedFiles = append(analysis.ExposedFiles, ExposedFile{
					Path:        f.path,
					Type:        f.fileType,
					Description: f.description,
					Risk:        f.risk,
				})
				if evidence != "" {
					analysis.Evidence[f.path] = evidence
				}
				if f.risk > highestRisk {
					highestRisk = f.risk
				}
				totalChecked++
				checkState[f.path] = false
				mu.Unlock()
			}
		}(file)
	}

	wg.Wait()

	// Apply false positive detection
	if s.detectFalsePositives(analysis, len(commonFiles), totalChecked) {
		analysis.Risk = RiskLow
		analysis.Evidence["false_positive"] = "High detection rate suggests potential false positives. Manual verification recommended."
		return analysis, nil
	}

	analysis.Risk = highestRisk
	analysis.Checks = buildFileChecks(commonFiles, checkState)
	return analysis, nil
}

func buildFileChecks(files []FileSignature, state map[string]bool) []FileCheck {
	checks := make([]FileCheck, 0, len(files))
	for _, file := range files {
		passed, ok := state[file.path]
		if !ok {
			passed = true
		}
		checks = append(checks, FileCheck{
			Path:   file.path,
			Passed: passed,
		})
	}
	return checks
}

func (s *Scanner) validateFile(ctx context.Context, domain string, file FileSignature) (bool, string) {
	resp, err := s.fetch(ctx, domain+file.path, requestOptions{})
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, ""
	}

	// Limit read size to prevent memory exhaustion
	content, err := io.ReadAll(io.LimitReader(resp.Body, MaxContentLength))
	if err != nil {
		return false, ""
	}

	if file.fileType != "Directory" && isLikelyHTML(content) {
		return false, ""
	}

	if !file.validate(content) {
		return false, ""
	}

	// Collect evidence based on file type
	evidence := s.collectEvidence(file, content)
	return true, evidence
}

func (s *Scanner) detectFalsePositives(analysis *FileExposureAnalysis, totalFiles, filesFound int) bool {
	if len(analysis.ExposedFiles) > MaxExposedFiles {
		return true
	}

	ratio := float64(filesFound) / float64(totalFiles)
	return ratio >= SuspiciousRatio
}

func (s *Scanner) collectEvidence(file FileSignature, content []byte) string {
	switch file.fileType {
	case "VCS":
		return "Repository information exposed. First 40 bytes: " + string(content[:40])
	case "Config":
		// Sanitize sensitive data before logging
		sanitized := s.sanitizeConfig(content)
		return "Configuration file exposed. Sample content: " + sanitized
	default:
		return "File exposed and validated."
	}
}

func (s *Scanner) sanitizeConfig(content []byte) string {
	// Remove potential sensitive data before logging
	lines := strings.Split(string(content), "\n")
	var sanitized []string
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "password") ||
			strings.Contains(strings.ToLower(line), "secret") ||
			strings.Contains(strings.ToLower(line), "key") {
			sanitized = append(sanitized, "[REDACTED]")
		} else {
			sanitized = append(sanitized, line)
		}
	}
	// Return first few lines only
	if len(sanitized) > 5 {
		sanitized = sanitized[:5]
	}
	return strings.Join(sanitized, "\n")
}

func isLikelyEnvFile(content []byte) bool {
	if isLikelyHTML(content) {
		return false
	}

	lines := strings.Split(string(content), "\n")
	validLines := 0
	checkedLines := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		checkedLines++
		if isEnvAssignment(trimmed) {
			validLines++
		}
	}

	if checkedLines == 0 {
		return false
	}

	return validLines > 0 && float64(validLines)/float64(checkedLines) >= 0.6
}

func isEnvAssignment(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	}
	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return false
	}

	key := strings.TrimSpace(parts[0])
	return isValidEnvKey(key)
}

func isValidEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for i := 0; i < len(key); i++ {
		ch := key[i]
		if i == 0 {
			if !(ch == '_' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')) {
				return false
			}
			continue
		}
		if !(ch == '_' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')) {
			return false
		}
	}
	return true
}

func isLikelyShellHistory(content []byte) bool {
	if isLikelyHTML(content) {
		return false
	}

	lines := strings.Split(string(content), "\n")
	commands := 0
	checked := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		checked++
		if looksLikeShellCommand(trimmed) {
			commands++
		}
	}

	if checked == 0 {
		return false
	}

	return commands > 0 && float64(commands)/float64(checked) >= 0.4
}

func looksLikeShellCommand(line string) bool {
	if strings.HasPrefix(line, "#") {
		return false
	}

	tokens := strings.Fields(line)
	if len(tokens) == 0 {
		return false
	}

	first := tokens[0]
	common := []string{
		"cd", "ls", "cat", "grep", "rg", "find", "curl", "wget", "git",
		"ssh", "scp", "cp", "mv", "rm", "chmod", "chown", "sudo",
		"docker", "kubectl", "ps", "top", "tail", "head", "env", "export",
	}
	for _, cmd := range common {
		if first == cmd {
			return true
		}
	}

	for i := 0; i < len(first); i++ {
		ch := first[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '-' || ch == '_' || ch == '/' || ch == '.') {
			return false
		}
	}
	return true
}

func isLikelySubversionEntries(content []byte) bool {
	if isLikelyHTML(content) {
		return false
	}

	lower := strings.ToLower(string(content))
	if strings.Contains(lower, "svn") && strings.Contains(lower, "dir") {
		return true
	}

	return strings.Contains(lower, "svn") && strings.Contains(lower, "entries")
}

func isLikelyMercurialRevlog(content []byte) bool {
	if isLikelyHTML(content) {
		return false
	}
	if len(content) < 32 {
		return false
	}
	return bytes.Contains(content, []byte{0})
}

type fileExposureTask struct{}

func newFileExposureTask() ScanTask {
	return fileExposureTask{}
}

func (fileExposureTask) Name() string {
	return "fileExposure"
}

func (fileExposureTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	files, err := scanner.checkFileExposure(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.FileExposure = *files
	return nil
}
