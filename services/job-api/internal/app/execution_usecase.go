package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"

	"github.com/cockroachdb/errors"
)

//go:generate moq -out gen_execution_store_moq_test.go . ExecutionStore
type ExecutionStore interface {
	Executions() ExecutionRepository
	WithinExecutionTx(ctx context.Context, fn func(ctx context.Context, executions ExecutionRepository) error) error
}

type ExecutionUsecase struct {
	s ExecutionStore
	d OutboxDispatcher
}

func (u *ExecutionUsecase) RequestExecution(ctx context.Context, actor domain.Actor, job_id domain.JobId, trigger domain.ExecutionTrigger) (domain.Execution, error) {
	if !actor.Role.HasCapability(domain.CapJobWrite) {
		return domain.Execution{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobWrite)
	}
	var execution domain.Execution
	if err := u.s.WithinExecutionTx(ctx, func(ctx context.Context, execs ExecutionRepository) error {
		e, err := execs.CreateForCurrentJob(ctx, actor.ID, job_id, trigger)
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

func (u *ExecutionUsecase) GetExecution(ctx context.Context, actor domain.Actor, execution_id domain.ExecutionId) (domain.Execution, error) {
	if !actor.Role.HasCapability(domain.CapJobRead) {
		return domain.Execution{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobRead)
	}
	return u.s.Executions().FindById(ctx, actor.ID, execution_id)
}

func (u *ExecutionUsecase) ListExecutions(ctx context.Context, actor domain.Actor, job_id domain.JobId, page_size int32, page_token string) (ListExecutionsResult, error) {
	if !actor.Role.HasCapability(domain.CapJobRead) {
		return ListExecutionsResult{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobRead)
	}
	return u.s.Executions().ListByJob(ctx, actor.ID, job_id, page_size, page_token)
}

func NewExecutionUsecase(s ExecutionStore, d OutboxDispatcher) *ExecutionUsecase {
	return &ExecutionUsecase{s: s, d: d}
}
