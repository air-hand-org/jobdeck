package app

import (
	"context"

	"jobdeck/services/job-api/internal/domain"
)

type ListExecutionsResult struct {
	Executions    []domain.Execution
	NextPageToken string
}

//go:generate moq -out gen_execution_repository_moq_test.go . ExecutionRepository
type ExecutionRepository interface {
	CreateForCurrentJob(ctx context.Context, owner_id domain.UserId, job_id domain.JobId, trigger domain.ExecutionTrigger) (domain.Execution, error)

	FindById(ctx context.Context, owner_id domain.UserId, execution_id domain.ExecutionId) (domain.Execution, error)
	ListByJob(ctx context.Context, owner_id domain.UserId, job_id domain.JobId, page_size int32, page_token string) (ListExecutionsResult, error)
}
