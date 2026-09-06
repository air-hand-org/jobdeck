package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"

	"github.com/cockroachdb/errors"
)

// ExecutionStore provides execution repositories and transactions.
//
//go:generate moq -out gen_execution_store_moq_test.go . ExecutionStore
type ExecutionStore interface {
	Executions() ExecutionRepository
	WithinExecutionTx(ctx context.Context, fn func(ctx context.Context, executions ExecutionRepository) error) error
}

// ExecutionUsecase implements execution operations.
type ExecutionUsecase struct {
	s ExecutionStore
	d OutboxDispatcher
}

// RequestExecution creates an execution request.
func (u *ExecutionUsecase) RequestExecution(ctx context.Context, actor domain.Actor, jobID domain.JobId, trigger domain.ExecutionTrigger) (domain.Execution, error) {
	if !actor.Role.HasCapability(domain.CapJobWrite) {
		return domain.Execution{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobWrite)
	}
	var execution domain.Execution
	if err := u.s.WithinExecutionTx(ctx, func(ctx context.Context, execs ExecutionRepository) error {
		e, err := execs.CreateForCurrentJob(ctx, actor.ID, jobID, trigger)
		if err != nil {
			return err
		}
		execution = e
		return nil
	}); err != nil {
		return domain.Execution{}, err
	}
	if err := u.d.Trigger(ctx); err != nil {
		return domain.Execution{}, err
	}
	return execution, nil
}

// GetExecution retrieves an execution.
func (u *ExecutionUsecase) GetExecution(ctx context.Context, actor domain.Actor, executionID domain.ExecutionId) (domain.Execution, error) {
	if !actor.Role.HasCapability(domain.CapJobRead) {
		return domain.Execution{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobRead)
	}
	return u.s.Executions().FindById(ctx, actor.ID, executionID)
}

// ListExecutions lists executions for a job.
func (u *ExecutionUsecase) ListExecutions(ctx context.Context, actor domain.Actor, jobID domain.JobId, pageSize int32, pageToken string) (ListExecutionsResult, error) {
	if !actor.Role.HasCapability(domain.CapJobRead) {
		return ListExecutionsResult{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobRead)
	}
	return u.s.Executions().ListByJob(ctx, actor.ID, jobID, pageSize, pageToken)
}

// NewExecutionUsecase constructs an execution use case.
func NewExecutionUsecase(s ExecutionStore, d OutboxDispatcher) *ExecutionUsecase {
	return &ExecutionUsecase{s: s, d: d}
}
