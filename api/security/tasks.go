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
		newHeadersTask(),
		newRobotsTask(),
		newSecurityTxtTask(),
		newCertificateTask(),
		newAdminTask(),
		newSwaggerTask(),
		newInfrastructureTask(),
		newDNSTask(),
		newFileExposureTask(),
		newSecretExposureTask(),
		newAPITask(),
		newHealthTask(),
	}
}
