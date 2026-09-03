package app

import (
	"context"

	"jobdeck/services/job-api/internal/domain"
)

// ListExecutionsResult contains a page of executions.
type ListExecutionsResult struct {
	Executions    []domain.Execution
	NextPageToken string
}

// ExecutionRepository persists executions.
//
//go:generate moq -out gen_execution_repository_moq_test.go . ExecutionRepository
type ExecutionRepository interface {
	CreateForCurrentJob(ctx context.Context, ownerID domain.ActorId, jobID domain.JobId, trigger domain.ExecutionTrigger) (domain.Execution, error)

	FindById(ctx context.Context, ownerID domain.ActorId, executionID domain.ExecutionId) (domain.Execution, error)
	ListByJob(ctx context.Context, ownerID domain.ActorId, jobID domain.JobId, pageSize int32, pageToken string) (ListExecutionsResult, error)
}
