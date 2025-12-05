package security

import "context"

type apiTask struct{}

func newAPITask() ScanTask {
	return apiTask{}
}

func (apiTask) Name() string {
	return "apiExposure"
}

func (apiTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	apiExposure, err := scanner.checkAPIExposures(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.APIExposure = *apiExposure
	return nil
}
