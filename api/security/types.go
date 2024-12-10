package security

type SecurityReport struct {
	Headers     HeadersAnalysis     `json:"headers"`
	Certificate CertificateAnalysis `json:"certificate"`
	AdminPages  AdminPagesAnalysis  `json:"adminPages"`
	Swagger     SwaggerAnalysis     `json:"swagger"`
	Screenshot  string              `json:"screenshot"` // Base64 encoded screenshot
}

type HeadersAnalysis struct {
	Score  string   `json:"score"`
	Issues []string `json:"issues"`
	Passed []string `json:"passed"`
}

type CertificateAnalysis struct {
	Grade      string   `json:"grade"`
	ValidUntil string   `json:"validUntil"`
	Issuer     string   `json:"issuer"`
	Findings   []string `json:"findings"`
	Warnings   []string `json:"warnings"`
}

type AdminPagesAnalysis struct {
	Exposed         []string `json:"exposed"`
	Risk            string   `json:"risk"`
	Recommendations []string `json:"recommendations"`
}

type SwaggerAnalysis struct {
	Endpoints       []string `json:"endpoints"`
	Exposed         bool     `json:"exposed"`
	Risk            string   `json:"risk"`
	Recommendations []string `json:"recommendations"`
}

type SecurityHeaders struct {
	Score  string   `json:"score"`
	Issues []string `json:"issues"`
	Passed []string `json:"passed"`
}

type Certificate struct {
	Grade      string   `json:"grade"`
	ValidUntil string   `json:"validUntil"`
	Issuer     string   `json:"issuer"`
	Findings   []string `json:"findings"`
	Warnings   []string `json:"warnings"`
}

type AdminPages struct {
	Exposed         []string `json:"exposed"`
	Risk            string   `json:"risk"`
	Recommendations []string `json:"recommendations"`
}

type SwaggerDocs struct {
	Endpoints       []string `json:"endpoints"`
	Exposed         bool     `json:"exposed"`
	Risk            string   `json:"risk"`
	Recommendations []string `json:"recommendations"`
}
