package security

import (
	"context"
)

type robotsTask struct{}

func newRobotsTask() ScanTask {
	return robotsTask{}
}

func (robotsTask) Name() string {
	return "robotsTxt"
}

func (robotsTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	robotsTxt, err := scanner.checkRobotsTxt(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.RobotsTxt = *robotsTxt
	return nil
}
