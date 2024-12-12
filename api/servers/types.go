package servers

import (
	"time"

	"gorm.io/gorm"
)

type Server struct {
	gorm.Model
	URL            string    `json:"url"`
	Active         bool      `json:"active"`
	FollowRedirect bool      `json:"follow_redirect"`
	LastSuccess    time.Time `json:"last_success"`
	FailureCount   int       `json:"failure_count"`
	NextCheck      time.Time `json:"next_check"`
	AllowInsecure  bool      `json:"allow_insecure"`
	PingResults    []PingResult
}

type PingResult struct {
	gorm.Model
	ServerID           uint      `json:"server_id"`
	OrganizationName   string    `json:"organization_name"`
	StatusCode         int       `json:"status_code"`
	IP                 string    `json:"ip"`
	ResponseTime       float64   `json:"response_time_ms"`
	Error              string    `json:"error"`
	RedirectCount      int       `json:"redirect_count"`
	Timestamp          time.Time `json:"timestamp"`
	TLSValid           bool      `json:"tls_valid"`
	CertExpiryDate     time.Time `json:"cert_expiry_date"`
	CertIssuer         string    `json:"cert_issuer"`
	CertCommonName     string    `json:"cert_common_name"`
	InsecureSkipVerify bool      `json:"insecure_skip_verify"`
}
