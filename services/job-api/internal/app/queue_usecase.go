package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
	"time"

	"github.com/cockroachdb/errors"
)

//go:generate moq -out gen_queue_store_moq_test.go . ExecutionQueueStore
type ExecutionQueueStore interface {
	Queues() ExecutionQueueRepository
	WithinExecutionQueueTx(ctx context.Context, fn func(ctx context.Context, queues ExecutionQueueRepository) error) error
}

type ExecutionQueueUsecase struct {
	s ExecutionQueueStore
	d OutboxDispatcher
}

func (u *ExecutionQueueUsecase) ClaimExecution(ctx context.Context, actor domain.Actor, execution_id domain.ExecutionId) (domain.ClaimedExecution, error) {
	if !actor.Role.HasCapability(domain.CapQueueClaim) {
		return domain.ClaimedExecution{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapQueueClaim)
	}

	var execution domain.ClaimedExecution
	err := u.s.WithinExecutionQueueTx(ctx, func(ctx context.Context, queues ExecutionQueueRepository) error {
		res, err := queues.Claim(ctx, execution_id)
		if err != nil {
			return errors.Wrapf(err, "failed to enqueue %s", execution_id)
		}
		execution = res
		return nil
	})
	if err != nil {
		return domain.ClaimedExecution{}, err
	}
	return execution, nil
}

type ReportExecutionResultCommand struct {
	ExecutionId     domain.ExecutionId
	ExecutionResult domain.ExecutionResult
	LeaseToken      domain.LeaseToken
}

func (u *ExecutionQueueUsecase) ReportExecutionResult(ctx context.Context, actor domain.Actor, command ReportExecutionResultCommand) error {
	if !actor.Role.HasCapability(domain.CapQueueReport) {
		return errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapQueueReport)
	}
	var retry_exec *domain.Execution
	if err := u.s.WithinExecutionQueueTx(ctx, func(ctx context.Context, queues ExecutionQueueRepository) error {
		e, err := queues.GetReportTarget(ctx, command.ExecutionId, command.LeaseToken)
		if err != nil {
			return err
		}

		status, err := e.DetermineStatus(command.ExecutionResult)
		if err != nil {
			return err
		}

		e, err = queues.ReportResult(ctx, command.ExecutionId, status, command.LeaseToken, command.ExecutionResult)
		if err != nil {
			return err
		}

		retry := e.ShouldRetry()
		if !retry {
			return nil
		}

		next_run_at := e.NextRetryRunAt(time.Now())
		retry_e, err := queues.CreateRetryUnsuccessful(ctx, e.Id, next_run_at)
		if err != nil {
			return err
		}

		retry_exec = &retry_e

		return nil
	}); err != nil {
		return err
	}

	if retry_exec != nil {
		if err := u.d.Trigger(ctx); err != nil {
			return err
		}
	}
	return nil
}

func NewExecutionQueueUsecase(s ExecutionQueueStore, d OutboxDispatcher) *ExecutionQueueUsecase {
	return &ExecutionQueueUsecase{s: s, d: d}
}
