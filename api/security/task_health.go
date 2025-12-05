package security

import "context"

type healthTask struct{}

func newHealthTask() ScanTask {
	return healthTask{}
}

func (healthTask) Name() string {
	return "healthProbes"
}

func (healthTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	health, err := scanner.checkHealthProbes(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.HealthProbes = *health
	return nil
}
