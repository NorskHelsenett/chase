// scanner.go
package security

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultScannerTimeout        = 10 * time.Second
	defaultMaxConcurrentRequests = 8
)

type Scanner struct {
	timeout           time.Duration
	strictTransport   *http.Transport
	insecureTransport *http.Transport

	tlsMu     sync.Mutex
	tlsIssues []string
	tasks     []ScanTask
	version   string

	requestLimiter chan struct{}
}

type requestOptions struct {
	followRedirects *bool
}

func NewScanner(timeout time.Duration) *Scanner {
	if timeout <= 0 {
		timeout = defaultScannerTimeout
	}

	scanner := &Scanner{
		timeout:           timeout,
		strictTransport:   newTransport(false),
		insecureTransport: newTransport(true),
	}

	scanner.tasks = defaultScanTasks()
	scanner.requestLimiter = make(chan struct{}, defaultMaxConcurrentRequests)
	scanner.recalculateVersion()
	return scanner
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

func (s *Scanner) RegisterTask(task ScanTask) {
	s.tasks = append(s.tasks, task)
	s.recalculateVersion()
}

func (s *Scanner) Tasks() []ScanTask {
	return append([]ScanTask(nil), s.tasks...)
}

func (s *Scanner) SetMaxConcurrentRequests(n int) {
	if n <= 0 {
		n = 1
	}
	s.requestLimiter = make(chan struct{}, n)
}

func (s *Scanner) recalculateVersion() {
	names := make([]string, 0, len(s.tasks))
	for _, task := range s.tasks {
		names = append(names, task.Name())
	}
	sort.Strings(names)
	sum := sha256.Sum256([]byte(strings.Join(names, "|")))
	s.version = hex.EncodeToString(sum[:8])
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

func (s *Scanner) doRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	if s.requestLimiter == nil {
		s.requestLimiter = make(chan struct{}, defaultMaxConcurrentRequests)
	}
	s.requestLimiter <- struct{}{}
	defer func() { <-s.requestLimiter }()
	return client.Do(req)
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
	report.ScannerVersion = s.version

	req := ScanRequest{Domain: domain}

	for _, task := range s.tasks {
		task := task
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := task.Run(ctx, s, req, report); err != nil {
				report.addError(task.Name(), err)
			}
		}()
	}

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
