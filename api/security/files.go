// files.go
package security

import (
	"net/http"
	"sync"
)

func (s *Scanner) checkFileExposure(domain string) (*FileExposureAnalysis, error) {
	analysis := &FileExposureAnalysis{
		ExposedFiles: make([]ExposedFile, 0),
		Risk:         RiskLow,
	}

	commonFiles := []struct {
		path        string
		fileType    string
		description string
		risk        RiskLevel
	}{
		{"/robots.txt", "Config", "Robots exclusion file", RiskLow},
		{"/sitemap.xml", "Config", "Site map file", RiskLow},
		{"/.git/HEAD", "VCS", "Git repository exposure", RiskHigh},
		{"/.env", "Config", "Environment file exposure", RiskCritical},
		{"/backup.zip", "Backup", "Backup file exposure", RiskHigh},
		{"/wp-config.php", "Config", "WordPress configuration file", RiskHigh},
		{"/config.php", "Config", "PHP configuration file", RiskHigh},
		{"/.htaccess", "Config", "Apache configuration file", RiskMedium},
		{"/web.config", "Config", "IIS configuration file", RiskMedium},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	highestRisk := RiskLow

	for _, file := range commonFiles {
		wg.Add(1)
		go func(f struct {
			path        string
			fileType    string
			description string
			risk        RiskLevel
		}) {
			defer wg.Done()
			resp, err := s.client.Head(domain + f.path)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusForbidden {
				exposed := ExposedFile{
					Path:        f.path,
					Type:        f.fileType,
					Description: f.description,
					Risk:        f.risk,
				}

				mu.Lock()
				analysis.ExposedFiles = append(analysis.ExposedFiles, exposed)
				if f.risk > highestRisk {
					highestRisk = f.risk
				}
				mu.Unlock()
			}
		}(file)
	}

	wg.Wait()
	analysis.Risk = highestRisk

	return analysis, nil
}
