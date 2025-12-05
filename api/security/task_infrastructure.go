package security

import "context"

type infrastructureTask struct{}

func newInfrastructureTask() ScanTask {
	return infrastructureTask{}
}

func (infrastructureTask) Name() string {
	return "infrastructure"
}

func (infrastructureTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	infra, err := scanner.checkInfrastructure(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.Infrastructure = *infra
	return nil
}
