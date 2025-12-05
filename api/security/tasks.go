package security

import (
	"context"
)

// ScanTask represents a pluggable scanner module.
type ScanTask interface {
	Name() string
	Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error
}

// ScanRequest carries metadata for tasks.
type ScanRequest struct {
	Domain string
}

// ScanTaskFunc helper to wrap simple funcs.
type ScanTaskFunc struct {
	name string
	run  func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error
}

func (f ScanTaskFunc) Name() string {
	return f.name
}

func (f ScanTaskFunc) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	return f.run(ctx, scanner, req, report)
}

func defaultScanTasks() []ScanTask {
	return []ScanTask{
		ScanTaskFunc{
			name: "robotsTxt",
			run: func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
				robotsTxt, err := scanner.checkRobotsTxt(ctx, req.Domain)
				if err != nil {
					return err
				}
				report.RobotsTxt = *robotsTxt
				return nil
			},
		},
		ScanTaskFunc{
			name: "securityTxt",
			run: func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
				securityTxt, err := scanner.checkSecurityTxt(ctx, req.Domain)
				if err != nil {
					return err
				}
				report.SecurityTxt = *securityTxt
				return nil
			},
		},
		ScanTaskFunc{
			name: "headers",
			run: func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
				headers, err := scanner.checkSecurityHeaders(ctx, req.Domain)
				if err != nil {
					return err
				}
				report.Headers = *headers
				return nil
			},
		},
		ScanTaskFunc{
			name: "certificate",
			run: func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
				cert, err := scanner.checkCertificate(ctx, req.Domain)
				if err != nil {
					return err
				}
				report.Certificate = *cert
				return nil
			},
		},
		ScanTaskFunc{
			name: "adminPages",
			run: func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
				admin, err := scanner.checkAdminPages(ctx, req.Domain)
				if err != nil {
					return err
				}
				report.AdminPages = *admin
				return nil
			},
		},
		ScanTaskFunc{
			name: "swagger",
			run: func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
				swagger, err := scanner.checkSwaggerDocs(ctx, req.Domain)
				if err != nil {
					return err
				}
				report.Swagger = *swagger
				return nil
			},
		},
		ScanTaskFunc{
			name: "infrastructure",
			run: func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
				infra, err := scanner.checkInfrastructure(ctx, req.Domain)
				if err != nil {
					return err
				}
				report.Infrastructure = *infra
				return nil
			},
		},
		ScanTaskFunc{
			name: "dns",
			run: func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
				dns, err := scanner.checkDNS(ctx, req.Domain)
				if err != nil {
					return err
				}
				report.DNSRecords = *dns
				return nil
			},
		},
		ScanTaskFunc{
			name: "files",
			run: func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
				files, err := scanner.checkFileExposure(ctx, req.Domain)
				if err != nil {
					return err
				}
				report.FileExposure = *files
				return nil
			},
		},
		ScanTaskFunc{
			name: "apiExposure",
			run: func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
				api, err := scanner.checkAPIExposures(ctx, req.Domain)
				if err != nil {
					return err
				}
				report.APIExposure = *api
				return nil
			},
		},
		ScanTaskFunc{
			name: "healthProbes",
			run: func(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
				health, err := scanner.checkHealthProbes(ctx, req.Domain)
				if err != nil {
					return err
				}
				report.HealthProbes = *health
				return nil
			},
		},
	}
}
