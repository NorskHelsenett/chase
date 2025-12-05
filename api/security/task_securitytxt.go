package security

import (
	"context"
)

type securityTxtTask struct{}

func newSecurityTxtTask() ScanTask {
	return securityTxtTask{}
}

func (securityTxtTask) Name() string {
	return "securityTxt"
}

func (securityTxtTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	securityTxt, err := scanner.checkSecurityTxt(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.SecurityTxt = *securityTxt
	return nil
}
