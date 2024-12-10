// scanner.go
package security

import (
	"crypto/tls"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Scanner struct {
	client *http.Client
}

func NewScanner() *Scanner {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 10,
	}

	return &Scanner{client: client}
}

func (s *Scanner) ScanWebsite(domain string) (*SecurityReport, error) {
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}

	var wg sync.WaitGroup
	report := &SecurityReport{
		ScanTimestamp: time.Now(),
		TargetURL:     domain,
		ScanErrors:    make([]ScanError, 0),
	}

	// Run all scans concurrently
	wg.Add(9)

	go func() {
		defer wg.Done()
		if robotsTxt, err := s.checkRobotsTxt(domain); err != nil {
			report.addError("headers", err)
		} else {
			report.RobotsTxt = *robotsTxt
		}
	}()

	go func() {
		defer wg.Done()
		if securityTxt, err := s.checkSecurityTxt(domain); err != nil {
			report.addError("headers", err)
		} else {
			report.SecurityTxt = *securityTxt
		}
	}()

	go func() {
		defer wg.Done()
		if headers, err := s.checkSecurityHeaders(domain); err != nil {
			report.addError("headers", err)
		} else {
			report.Headers = *headers
		}
	}()

	go func() {
		defer wg.Done()
		if cert, err := s.checkCertificate(domain); err != nil {
			report.addError("certificate", err)
		} else {
			report.Certificate = *cert
		}
	}()

	go func() {
		defer wg.Done()
		if admin, err := s.checkAdminPages(domain); err != nil {
			report.addError("adminPages", err)
		} else {
			report.AdminPages = *admin
		}
	}()

	go func() {
		defer wg.Done()
		if swagger, err := s.checkSwaggerDocs(domain); err != nil {
			report.addError("swagger", err)
		} else {
			report.Swagger = *swagger
		}
	}()

	go func() {
		defer wg.Done()
		if infra, err := s.checkInfrastructure(domain); err != nil {
			report.addError("infrastructure", err)
		} else {
			report.Infrastructure = *infra
		}
	}()

	go func() {
		defer wg.Done()
		if dns, err := s.checkDNS(domain); err != nil {
			report.addError("dns", err)
		} else {
			report.DNSRecords = *dns
		}
	}()

	go func() {
		defer wg.Done()
		if files, err := s.checkFileExposure(domain); err != nil {
			report.addError("files", err)
		} else {
			report.FileExposure = *files
		}
	}()

	wg.Wait()
	return report, nil
}

func (r *SecurityReport) addError(component string, err error) {
	r.ScanErrors = append(r.ScanErrors, ScanError{
		Component: component,
		Error:     err.Error(),
		Timestamp: time.Now(),
	})
}
