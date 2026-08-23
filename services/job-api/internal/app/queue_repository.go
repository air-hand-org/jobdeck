package app

import (
	"context"
	"time"

	"jobdeck/services/job-api/internal/domain"
)

//go:generate moq -out gen_queue_repository_moq_test.go . ExecutionQueueRepository
type ExecutionQueueRepository interface {
	CreateRetryUnsuccessful(ctx context.Context, execution_id domain.ExecutionId, next_run_at time.Time) (domain.Execution, error)

	Claim(ctx context.Context, execution_id domain.ExecutionId) (domain.ClaimedExecution, error)

	GetReportTarget(ctx context.Context, execution_id domain.ExecutionId, lease_token domain.LeaseToken) (domain.Execution, error)
	ReportResult(ctx context.Context, execution_id domain.ExecutionId, status domain.ExecutionStatus, lease_token domain.LeaseToken, result domain.ExecutionResult) (domain.Execution, error)
}
