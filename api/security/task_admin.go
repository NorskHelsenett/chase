package security

import "context"

type adminTask struct{}

func newAdminTask() ScanTask {
	return adminTask{}
}

func (adminTask) Name() string {
	return "adminPages"
}

func (adminTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	admin, err := scanner.checkAdminPages(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.AdminPages = *admin
	return nil
}
