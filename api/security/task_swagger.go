package security

import "context"

type swaggerTask struct{}

func newSwaggerTask() ScanTask {
	return swaggerTask{}
}

func (swaggerTask) Name() string {
	return "swagger"
}

func (swaggerTask) Run(ctx context.Context, scanner *Scanner, req ScanRequest, report *SecurityReport) error {
	swagger, err := scanner.checkSwaggerDocs(ctx, req.Domain)
	if err != nil {
		return err
	}
	report.Swagger = *swagger
	return nil
}
