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
				lines := strings.Split(string(content), "\n")
				envVarCount := 0
				for _, line := range lines {
					if strings.Contains(line, "=") {
						envVarCount++
					}
				}
				return envVarCount > 0
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
				return bytes.Contains(content, []byte("svn")) || bytes.Contains(content, []byte("dir"))
			},
		},
		{
			path:        "/.hg/store/00manifest.i",
			fileType:    "VCS",
			description: "Mercurial repository exposed",
			risk:        RiskHigh,
			validate: func(content []byte) bool {
				return len(content) > 0
			},
		},
		{
			path:        "/.DS_Store",
			fileType:    "Metadata",
			description: "macOS Finder metadata exposed",
			risk:        RiskLow,
			validate: func(content []byte) bool {
				return len(content) > 0
			},
		},
		{
			path:        "/.bash_history",
			fileType:    "Secrets",
			description: "Shell history exposed",
			risk:        RiskCritical,
			validate: func(content []byte) bool {
				return bytes.Contains(content, []byte("\n"))
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
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	highestRisk := RiskLow
	totalChecked := 0

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
	return analysis, nil
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
