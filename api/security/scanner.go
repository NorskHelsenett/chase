// scanner.go
package security

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
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
	options        ScannerOptions
}

type requestOptions struct {
	followRedirects *bool
	method          string
	headers         http.Header
	body            []byte
}

type ScannerOptions struct {
	EnabledTasks          []string
	MaxConcurrentRequests int
}

func ScannerOptionsFromEnv() ScannerOptions {
	opts := ScannerOptions{}

	if raw := os.Getenv("SCANNER_TASKS"); raw != "" {
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				opts.EnabledTasks = append(opts.EnabledTasks, part)
			}
		}
	}

	if raw := os.Getenv("SCANNER_MAX_CONCURRENCY"); raw != "" {
		if value, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && value > 0 {
			opts.MaxConcurrentRequests = value
		}
	}

	return opts
}

func NewScanner(timeout time.Duration) *Scanner {
	return NewScannerWithOptions(timeout, ScannerOptionsFromEnv())
}

func NewScannerWithOptions(timeout time.Duration, opts ScannerOptions) *Scanner {
	if timeout <= 0 {
		timeout = defaultScannerTimeout
	}

	scanner := &Scanner{
		timeout:           timeout,
		strictTransport:   newTransport(false),
		insecureTransport: newTransport(true),
		options:           opts,
	}

	scanner.requestLimiter = make(chan struct{}, defaultMaxConcurrentRequests)
	scanner.applyOptions()
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
	for _, existing := range s.tasks {
		if existing.Name() == task.Name() {
			return
		}
	}
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

func (s *Scanner) applyOptions() {
	catalog := defaultScanTasks()
	selected := make([]ScanTask, 0, len(catalog))

	if len(s.options.EnabledTasks) > 0 {
		allowed := make(map[string]struct{}, len(s.options.EnabledTasks))
		for _, name := range s.options.EnabledTasks {
			allowed[strings.TrimSpace(name)] = struct{}{}
		}
		for _, task := range catalog {
			if _, ok := allowed[task.Name()]; ok {
				selected = append(selected, task)
			}
		}
	}

	if len(selected) == 0 {
		selected = catalog
	}

	for _, task := range selected {
		s.RegisterTask(task)
	}

	if s.options.MaxConcurrentRequests > 0 {
		s.SetMaxConcurrentRequests(s.options.MaxConcurrentRequests)
	}
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

	method := http.MethodGet
	if opts.method != "" {
		method = opts.method
	}

	buildRequest := func() (*http.Request, error) {
		var body io.Reader
		if len(opts.body) > 0 {
			body = bytes.NewReader(opts.body)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, err
		}

		if opts.headers != nil {
			for key, values := range opts.headers {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}
		}
		return req, nil
	}

	req, err := buildRequest()
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

	req, err = buildRequest()
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
		Error:     sanitizeScanError(err.Error()),
		Timestamp: time.Now(),
	})
}

func sanitizeScanError(message string) string {
	if message == "" {
		return message
	}
	re := regexp.MustCompile(`https?://[^\s"']+`)
	return re.ReplaceAllString(message, "[REDACTED_URL]")
}
