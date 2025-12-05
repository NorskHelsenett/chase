package security

import "context"

type certificateTask struct{}

func newCertificateTask() ScanTask {
	return certificateTask{}
}

func (certificateTask) Name() string {
	return "certificate"
}

func (certificateTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	cert, err := scanner.checkCertificate(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.Certificate = *cert
	return nil
}
