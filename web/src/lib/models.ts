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
  security: SecurityReport;
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