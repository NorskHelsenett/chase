package security

import "time"

type SecurityReport struct {
	ScanTimestamp  time.Time              `json:"scanTimestamp"`
	TargetURL      string                 `json:"targetUrl"`
	ScannerVersion string                 `json:"scannerVersion"`
	Headers        HeadersAnalysis        `json:"headers"`
	Certificate    CertificateAnalysis    `json:"certificate"`
	AdminPages     AdminPagesAnalysis     `json:"adminPages"`
	Swagger        SwaggerAnalysis        `json:"swagger"`
	Screenshot     string                 `json:"screenshot"`
	Infrastructure InfrastructureAnalysis `json:"infrastructure"`
	RobotsTxt      RobotsAnalysis         `json:"robotsTxt"`
	SecurityTxt    SecurityTxtAnalysis    `json:"securityTxt"`
	Emails         []string               `json:"emails"`
	DNSRecords     DNSAnalysis            `json:"dnsRecords"`
	FileExposure   FileExposureAnalysis   `json:"fileExposure"`
	SecretExposure SecretExposureAnalysis `json:"secretExposure"`
	ScanErrors     []ScanError            `json:"scanErrors"`
	APIExposure    APIExposureAnalysis    `json:"apiExposure"`
	HealthProbes   HealthProbeAnalysis    `json:"healthProbes"`
}

type RobotsAnalysis struct {
	Exists      bool      `json:"exists"`
	Content     string    `json:"content"`
	ContentType string    `json:"-"` // To verify it's text/plain
	Findings    []Finding `json:"findings"`
	Risk        RiskLevel `json:"risk"`
}

type SecurityTxtAnalysis struct {
	Exists          bool      `json:"exists"`
	Content         string    `json:"content"`
	ContentType     string    `json:"contentType"`
	ValidSignature  bool      `json:"validSignature"`
	Expiration      time.Time `json:"expiration"`
	Contacts        []string  `json:"contacts"`
	Canonical       []string  `json:"canonical"`
	Encryptions     []string  `json:"encryptions"`
	Acknowledgments []string  `json:"acknowledgments"`
	Findings        []Finding `json:"findings"`
	Risk            RiskLevel `json:"risk"`
}

type ScanError struct {
	Component string    `json:"component"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

type DNSAnalysis struct {
	ARecords     []string `json:"aRecords"`
	MXRecords    []string `json:"mxRecords"`
	TXTRecords   []string `json:"txtRecords"`
	NSRecords    []string `json:"nsRecords"`
	CNAMERecords []string `json:"cnameRecords"`
	CAARecords   []string `json:"caaRecords"`
	Findings     []string `json:"findings"`
}

type FileExposureAnalysis struct {
	ExposedFiles []ExposedFile     `json:"exposedFiles"`
	Risk         RiskLevel         `json:"risk"`
	Evidence     map[string]string `json:"evidence"`
}

type SecretExposureAnalysis struct {
	Findings []Finding  `json:"findings"`
	Risk     RiskLevel  `json:"risk"`
	Sources  []string   `json:"sources,omitempty"`
}

type ExposedFile struct {
	Path        string    `json:"path"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Risk        RiskLevel `json:"risk"`
}

type RiskLevel string

const (
	RiskCritical RiskLevel = "CRITICAL"
	RiskHigh     RiskLevel = "HIGH"
	RiskMedium   RiskLevel = "MEDIUM"
	RiskLow      RiskLevel = "LOW"
	RiskInfo     RiskLevel = "INFO"
)

type Finding struct {
	Description string    `json:"description"`
	Risk        RiskLevel `json:"risk"`
	Evidence    string    `json:"evidence"`
	Mitigation  string    `json:"mitigation"`
}

type WhoisInfo struct {
	DomainName      string `json:"domainName"`
	Registrar       string `json:"registrar"`
	CreationDate    string `json:"creationDate"`
	ExpirationDate  string `json:"expirationDate"`
	LastUpdatedDate string `json:"lastUpdatedDate"`
}

// Existing types updated with RiskLevel
type InfrastructureAnalysis struct {
	IPAddress     string               `json:"ip"`
	IPv6Addresses []string             `json:"ipv6,omitempty"`
	HTTPStatus    string               `json:"status"`
	HTTPVersion   string               `json:"httpVersion"`
	Server        string               `json:"server"`
	CDNProvider   string               `json:"cdnProvider,omitempty"`
	Technology    []TechnologyAnalysis `json:"tech"`
	Risk          RiskLevel            `json:"risk"`
	Findings      []Finding            `json:"findings"`
}

type TechnologyAnalysis struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type HeadersAnalysis struct {
	Score          string    `json:"score"`
	Title          string    `json:"title"`
	Issues         []Finding `json:"issues"` // Changed to Finding type
	CookieFindings []Finding `json:"cookieFindings,omitempty"`
	CORSFindings   []Finding `json:"corsFindings,omitempty"`
	Passed         []string  `json:"passed"`
	Risk           RiskLevel `json:"risk"`
}

type CertificateAnalysis struct {
	Grade              string    `json:"grade"`
	ValidFrom          time.Time `json:"validFrom"`
	ValidUntil         time.Time `json:"validUntil"`
	Issuer             string    `json:"issuer"`
	Organization       string    `json:"organization"`
	SubjectDNS         []string  `json:"subjectDNS"`
	SerialNumber       string    `json:"serialNumber"`
	SignatureAlg       string    `json:"signatureAlgorithm"`
	PublicKeyType      string    `json:"publicKeyType"`
	PublicKeyBits      int       `json:"publicKeyBits"`
	Findings           []Finding `json:"findings"`
	Warnings           []Finding `json:"warnings"`
	Risk               RiskLevel `json:"risk"`
	TLSVersions        []string  `json:"tlsVersions"`
	SupportedCiphers   []Cipher  `json:"supportedCiphers"`
	CTEnabled          bool      `json:"ctEnabled"`
	RevocationStatus   string    `json:"revocationStatus"`
	NegotiatedProtocol string    `json:"alpnProtocol"`
	PreferredCipher    string    `json:"preferredCipher"`
}

type Cipher struct {
	Name        string `json:"name"`
	KeyExchange string `json:"keyExchange"`
	Strength    int    `json:"strength"`
	Forward     bool   `json:"forwardSecrecy"`
	Weak        bool   `json:"weak"`
}

type AdminPagesAnalysis struct {
	Exposed         []string          `json:"exposed"`
	Risk            RiskLevel         `json:"risk"`
	Findings        []Finding         `json:"findings"`
	Recommendations []string          `json:"recommendations"`
	Evidence        map[string]string `json:"evidence"`
}

type SwaggerAnalysis struct {
	Endpoints       []string  `json:"endpoints"`
	Exposed         bool      `json:"exposed"`
	Risk            RiskLevel `json:"risk"`
	Findings        []Finding `json:"findings"`
	Recommendations []string  `json:"recommendations"`
}

type APIExposureAnalysis struct {
	Endpoints []string  `json:"endpoints"`
	Risk      RiskLevel `json:"risk"`
	Findings  []Finding `json:"findings"`
}

type HealthProbeAnalysis struct {
	Paths    map[string]int `json:"paths"`
	Risk     RiskLevel      `json:"risk"`
	Findings []Finding      `json:"findings"`
}
