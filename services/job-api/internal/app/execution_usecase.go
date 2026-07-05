package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
)

//go:generate moq -out gen_execution_store_moq_test.go . ExecutionStore
type ExecutionStore interface {
	Executions() ExecutionRepository
	WithinExecutionTx(ctx context.Context, fn func(ctx context.Context, executions ExecutionRepository) error) error
}

type ExecutionUsecase struct {
	s ExecutionStore
}

func (u *ExecutionUsecase) RequestExecution(ctx context.Context, owner_id domain.UserId, job_id domain.JobId, trigger domain.ExecutionTrigger) (domain.Execution, error) {
	var execution domain.Execution
	err := u.s.WithinExecutionTx(ctx, func(ctx context.Context, execs ExecutionRepository) error {
		e, err := execs.CreateForCurrentJob(ctx, owner_id, job_id, trigger)
		if err != nil {
			return err
		}
		execution = e
		return nil
	})
	return execution, err
}

func (u *ExecutionUsecase) GetExecution(ctx context.Context, owner_id domain.UserId, execution_id domain.ExecutionId) (domain.Execution, error) {
	return u.s.Executions().FindById(ctx, owner_id, execution_id)
}

func (u *ExecutionUsecase) ListExecutions(ctx context.Context, owner_id domain.UserId, job_id domain.JobId, page_size int32, page_token string) (ListExecutionsResult, error) {
	return u.s.Executions().ListByJob(ctx, owner_id, job_id, page_size, page_token)
}

func NewExecutionUsecase(s ExecutionStore) *ExecutionUsecase {
	return &ExecutionUsecase{s: s}
}
