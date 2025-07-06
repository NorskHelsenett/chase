export interface BaseModel {
	ID: number;
	CreatedAt: Date;
	UpdatedAt: Date;
	DeletedAt?: Date;
}

export interface Server extends BaseModel {
	url: string;
	active: boolean;
	follow_redirect: boolean;
	failure_count: number;
	next_check: Date;
	allow_insecure: boolean;
	expected_status: number;
	comment: string;
	ping_results: PingResult[];
	security?: SecurityReport; // Make optional for backward compatibility
	update_interval: number;
	certificate?: Certificate;
	// New fields from server endpoint
	security_risk_level?: string;
	security_description?: string;
	security_scan_time?: string;
	header_score?: string;
	cert_score?: string;
	admin_risk?: string;
	api_risk?: string;
}

interface Finding {
	description: string;
	risk: RiskLevel;
	evidence: string;
	mitigation: string;
}

interface Cipher {
	name: string;
	keyExchange: string;
	strength: number;
	forward: boolean;
	weak: boolean;
}

type RiskLevel = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';

type Grade = 'A+' | 'A' | 'B' | 'C' | 'D' | 'F';

interface Certificate {
	grade: Grade;
	validUntil: string; // ISO 8601 date string
	issuer: string;
	organization: string;
	findings: Finding[];
	warnings: Finding[];
	risk: RiskLevel;
	tlsVersions: string[];
	supportedCiphers: Cipher[] | null;
	ctEnabled: boolean;
	revocationStatus: string;
}

export interface PingResult extends BaseModel {
	server_id: number;
	organization_name: string;
	status_code: number;
	ip: string;
	response_time_ms: number;
	error: string;
	redirect_count: number;
	timestamp: Date;
	tls_valid: boolean;
	cert_expiry_date: Date;
	cert_issuer: string;
	cert_common_name: string;
}

export interface SecurityReport {
	serverUrl: string;
	createdAt: Date;
	riskLevel: string;
	headerRisk: 'A+' | 'A' | 'B' | 'C' | 'D' | 'F' | '';
	certRisk: 'A+' | 'A' | 'B' | 'C' | 'D' | 'F' | '';
	adminRisk: 'critical' | 'high' | 'medium' | 'low' | '';
	apiRisk: 'critical' | 'high' | 'medium' | 'low' | '';
}

export interface Stats {
	up: number;
	down: number;
	criticalRisks: number;
	highRisks: number;
}
