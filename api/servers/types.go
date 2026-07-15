package servers

import (
	"time"

	"gorm.io/gorm"
)

type Server struct {
	gorm.Model
	URL                string    `json:"url" gorm:"uniqueIndex:idx_server_url"`
	Active             bool      `json:"active" gorm:"index:idx_active_next_check"`
	FollowRedirect     bool      `json:"follow_redirect"`
	NextCheck          time.Time `json:"next_check" gorm:"index:idx_active_next_check"`
	AllowInsecure      bool      `json:"allow_insecure"`
	ExpectedStatusCode int       `json:"expected_status"`
	Comment            string    `json:"comment"`
	// FirstSeen is when this host was first observed. It drives the "New" filter
	// (hosts first seen < 30 days ago). Unlike CreatedAt, it can be set explicitly
	// on import so it survives CSV round-trips and database migrations.
	FirstSeen        time.Time    `json:"first_seen" gorm:"index"`
	UpdateInterval   int          `json:"update_interval" gorm:"default:15"` // in minutes
	Favicon          string       `json:"favicon,omitempty"`
	FaviconMime      string       `json:"-" gorm:"default:''"`
	FaviconData      []byte       `json:"-"`
	FaviconFetchedAt *time.Time   `json:"-"`
	SiteTitle        string       `json:"site_title,omitempty"`
	SiteDescription  string       `json:"site_description,omitempty"`
	OGImage          string       `json:"og_image,omitempty"`
	PingResults      []PingResult `gorm:"foreignKey:ServerID;references:ID;OnDelete:CASCADE" json:"ping_results"`
	// Security report metadata (not stored in database)
	SecurityRiskLevel   string    `json:"security_risk_level,omitempty" gorm:"-"`
	SecurityDescription string    `json:"security_description,omitempty" gorm:"-"`
	SecurityScanTime    time.Time `json:"security_scan_time,omitempty" gorm:"-"`
	// Additional security details
	HeaderScore string `json:"header_score,omitempty" gorm:"-"`
	CertScore   string `json:"cert_score,omitempty" gorm:"-"`
	AdminRisk   string `json:"admin_risk,omitempty" gorm:"-"`
	APIRisk     string `json:"api_risk,omitempty" gorm:"-"`
}

// BeforeCreate defaults FirstSeen to the current time so every server has a
// stable "first observed" timestamp. Import paths may set FirstSeen explicitly
// (e.g. from a CSV round-trip, or to suppress the "New" badge for old hosts) to
// preserve the original value.
func (s *Server) BeforeCreate(tx *gorm.DB) error {
	if s.FirstSeen.IsZero() {
		s.FirstSeen = time.Now()
	}
	return nil
}

type PingResult struct {
	gorm.Model
	ServerID     uint        `json:"server_id" gorm:"index:idx_server_timestamp"`
	Server       Server      `gorm:"foreignKey:ServerID" json:"-"`
	StatusCode   int         `json:"status_code"`
	ResponseTime float64     `json:"response_time_ms"`
	Error        string      `json:"error"`
	Timestamp    time.Time   `json:"timestamp" gorm:"index:idx_server_timestamp"`
	DetailID     *uint       `json:"detail_id,omitempty" gorm:"index"`
	PingDetail   *PingDetail `gorm:"foreignKey:DetailID" json:"detail,omitempty"`

	// Transient — extracted during ping, used to update Server, not persisted
	siteMetadata SiteMetadata `gorm:"-" json:"-"`
}

// SiteMetadata holds lightweight metadata extracted from an HTML page's <head>.
type SiteMetadata struct {
	Favicon     string
	ManifestURL string
	Title       string
	Description string
	OGImage     string
}

// PingHourlySummary aggregates raw pings into one row per server per hour.
// Covers data between 7 days and 30 days old.
type PingHourlySummary struct {
	ID              uint      `gorm:"primaryKey"`
	ServerID        uint      `json:"server_id" gorm:"uniqueIndex:idx_hourly_server_hour"`
	Hour            time.Time `json:"hour" gorm:"uniqueIndex:idx_hourly_server_hour"`
	Total           int       `json:"total"`
	Successful      int       `json:"successful"`
	Failed          int       `json:"failed"`
	AvgResponseTime float64   `json:"avg_response_time_ms"`
	MinResponseTime float64   `json:"min_response_time_ms"`
	MaxResponseTime float64   `json:"max_response_time_ms"`
}

// PingDailySummary aggregates hourly summaries into one row per server per day.
// Covers data older than 30 days. Kept indefinitely (tiny footprint).
type PingDailySummary struct {
	ID              uint      `gorm:"primaryKey"`
	ServerID        uint      `json:"server_id" gorm:"uniqueIndex:idx_daily_server_date"`
	Date            time.Time `json:"date" gorm:"uniqueIndex:idx_daily_server_date;type:date"`
	Total           int       `json:"total"`
	Successful      int       `json:"successful"`
	Failed          int       `json:"failed"`
	AvgResponseTime float64   `json:"avg_response_time_ms"`
	MinResponseTime float64   `json:"min_response_time_ms"`
	MaxResponseTime float64   `json:"max_response_time_ms"`
}

type PingDetail struct {
	gorm.Model
	OrganizationName string    `json:"organization_name"`
	IP               string    `json:"ip" gorm:"index"`
	RedirectCount    int       `json:"redirect_count"`
	TLSValid         bool      `json:"tls_valid"`
	CertExpiryDate   time.Time `json:"cert_expiry_date" gorm:"index"`
	CertIssuer       string    `json:"cert_issuer"`
	CertCommonName   string    `json:"cert_common_name" gorm:"index"`
}
