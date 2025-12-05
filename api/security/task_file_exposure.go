package security

import "context"

type fileExposureTask struct{}

func newFileExposureTask() ScanTask {
	return fileExposureTask{}
}

func (fileExposureTask) Name() string {
	return "fileExposure"
}

func (fileExposureTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	files, err := scanner.checkFileExposure(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.FileExposure = *files
	return nil
}
