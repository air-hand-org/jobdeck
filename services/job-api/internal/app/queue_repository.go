package app

import (
	"context"
	"time"

	"jobdeck/services/job-api/internal/domain"
)

// ExecutionQueueRepository provides queue operations.
//
//go:generate moq -out gen_queue_repository_moq_test.go . ExecutionQueueRepository
type ExecutionQueueRepository interface {
	CreateRetryUnsuccessful(ctx context.Context, executionID domain.ExecutionId, nextRunAt time.Time) (domain.Execution, error)

	Claim(ctx context.Context, executionID domain.ExecutionId) (domain.ClaimedExecution, error)

	GetReportTarget(ctx context.Context, executionID domain.ExecutionId, leaseToken domain.LeaseToken) (domain.Execution, error)
	ReportResult(ctx context.Context, executionID domain.ExecutionId, status domain.ExecutionStatus, leaseToken domain.LeaseToken, result domain.ExecutionResult) (domain.Execution, error)
}
