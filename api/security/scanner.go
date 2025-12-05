// scanner.go
package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultScannerTimeout = 10 * time.Second

type Scanner struct {
	timeout           time.Duration
	strictTransport   *http.Transport
	insecureTransport *http.Transport

	tlsMu     sync.Mutex
	tlsIssues []string
}

type requestOptions struct {
	followRedirects *bool
}

func NewScanner(timeout time.Duration) *Scanner {
	if timeout <= 0 {
		timeout = defaultScannerTimeout
	}

	return &Scanner{
		timeout:           timeout,
		strictTransport:   newTransport(false),
		insecureTransport: newTransport(true),
	}
}

func newTransport(insecure bool) *http.Transport {
	return &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        50,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ForceAttemptHTTP2:   true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func (s *Scanner) client(followRedirects bool, insecure bool) *http.Client {
	client := &http.Client{
		Transport: s.strictTransport,
		Timeout:   s.timeout,
	}
	if insecure {
		client.Transport = s.insecureTransport
	}
	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return client
}

func (s *Scanner) recordTLSIssue(url string, err error) {
	s.tlsMu.Lock()
	defer s.tlsMu.Unlock()
	s.tlsIssues = append(s.tlsIssues, fmt.Sprintf("%s: %v", url, err))
}

func (s *Scanner) appendTLSIssues(report *SecurityReport) {
	s.tlsMu.Lock()
	defer s.tlsMu.Unlock()
	for _, issue := range s.tlsIssues {
		report.addError("tls", errors.New(issue))
	}
}

func (s *Scanner) fetch(ctx context.Context, url string, opts requestOptions) (*http.Response, error) {
	followRedirects := true
	if opts.followRedirects != nil {
		followRedirects = *opts.followRedirects
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client(followRedirects, false).Do(req)
	if err == nil {
		return resp, nil
	}

	if !isTLSError(err) {
		return nil, err
	}

	s.recordTLSIssue(url, err)

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return s.client(followRedirects, true).Do(req)
}

func isTLSError(err error) bool {
	var unknownAuth *x509.UnknownAuthorityError
	var certInvalid *x509.CertificateInvalidError
	var hostnameErr x509.HostnameError
	var recordErr *tls.RecordHeaderError
	var opErr *net.OpError

	if errors.As(err, &unknownAuth) ||
		errors.As(err, &certInvalid) ||
		errors.As(err, &hostnameErr) ||
		errors.As(err, &recordErr) {
		return true
	}

	if errors.As(err, &opErr) && opErr != nil && opErr.Err != nil {
		return strings.Contains(strings.ToLower(opErr.Err.Error()), "certificate")
	}

	return false
}

func (s *Scanner) ScanWebsite(ctx context.Context, domain string) (*SecurityReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}

	var wg sync.WaitGroup
	report := &SecurityReport{
		ScanTimestamp: time.Now(),
		TargetURL:     domain,
		ScanErrors:    make([]ScanError, 0),
	}

	wg.Add(11)

	go func() {
		defer wg.Done()
		if robotsTxt, err := s.checkRobotsTxt(ctx, domain); err != nil {
			report.addError("headers", err)
		} else {
			report.RobotsTxt = *robotsTxt
		}
	}()

	go func() {
		defer wg.Done()
		if securityTxt, err := s.checkSecurityTxt(ctx, domain); err != nil {
			report.addError("headers", err)
		} else {
			report.SecurityTxt = *securityTxt
		}
	}()

	go func() {
		defer wg.Done()
		if headers, err := s.checkSecurityHeaders(ctx, domain); err != nil {
			report.addError("headers", err)
		} else {
			report.Headers = *headers
		}
	}()

	go func() {
		defer wg.Done()
		if cert, err := s.checkCertificate(ctx, domain); err != nil {
			report.addError("certificate", err)
		} else {
			report.Certificate = *cert
		}
	}()

	go func() {
		defer wg.Done()
		if admin, err := s.checkAdminPages(ctx, domain); err != nil {
			report.addError("adminPages", err)
		} else {
			report.AdminPages = *admin
		}
	}()

	go func() {
		defer wg.Done()
		if swagger, err := s.checkSwaggerDocs(ctx, domain); err != nil {
			report.addError("swagger", err)
		} else {
			report.Swagger = *swagger
		}
	}()

	go func() {
		defer wg.Done()
		if infra, err := s.checkInfrastructure(ctx, domain); err != nil {
			report.addError("infrastructure", err)
		} else {
			report.Infrastructure = *infra
		}
	}()

	go func() {
		defer wg.Done()
		if dns, err := s.checkDNS(ctx, domain); err != nil {
			report.addError("dns", err)
		} else {
			report.DNSRecords = *dns
		}
	}()

	go func() {
		defer wg.Done()
		if files, err := s.checkFileExposure(ctx, domain); err != nil {
			report.addError("files", err)
		} else {
			report.FileExposure = *files
		}
	}()

	go func() {
		defer wg.Done()
		if apis, err := s.checkAPIExposures(ctx, domain); err != nil {
			report.addError("api", err)
		} else {
			report.APIExposure = *apis
		}
	}()

	go func() {
		defer wg.Done()
		if health, err := s.checkHealthProbes(ctx, domain); err != nil {
			report.addError("health", err)
		} else {
			report.HealthProbes = *health
		}
	}()

	wg.Wait()
	s.appendTLSIssues(report)
	return report, nil
}

func (r *SecurityReport) addError(component string, err error) {
	if err == nil {
		return
	}
	r.ScanErrors = append(r.ScanErrors, ScanError{
		Component: component,
		Error:     err.Error(),
		Timestamp: time.Now(),
	})
}
