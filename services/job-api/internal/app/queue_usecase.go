package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
	"time"

	"github.com/cockroachdb/errors"
)

// ExecutionQueueStore provides queue repositories and transactions.
//
//go:generate moq -out gen_queue_store_moq_test.go . ExecutionQueueStore
type ExecutionQueueStore interface {
	Queues() ExecutionQueueRepository
	WithinExecutionQueueTx(ctx context.Context, fn func(ctx context.Context, queues ExecutionQueueRepository) error) error
}

// ExecutionQueueUsecase implements queue operations.
type ExecutionQueueUsecase struct {
	s ExecutionQueueStore
	d OutboxDispatcher
}

// ClaimExecution claims an execution for a worker.
func (u *ExecutionQueueUsecase) ClaimExecution(ctx context.Context, actor domain.Actor, executionID domain.ExecutionId) (domain.ClaimedExecution, error) {
	if !actor.Role.HasCapability(domain.CapQueueClaim) {
		return domain.ClaimedExecution{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapQueueClaim)
	}

	var execution domain.ClaimedExecution
	err := u.s.WithinExecutionQueueTx(ctx, func(ctx context.Context, queues ExecutionQueueRepository) error {
		res, err := queues.Claim(ctx, executionID)
		if err != nil {
			return errors.Wrapf(err, "failed to enqueue %s", executionID)
		}
		execution = res
		return nil
	})
	if err != nil {
		return domain.ClaimedExecution{}, err
	}
	return execution, nil
}

// ReportExecutionResultCommand contains a worker result and lease.
type ReportExecutionResultCommand struct {
	//nolint:revive // Retained to match the protobuf field name.
	ExecutionId     domain.ExecutionId
	ExecutionResult domain.ExecutionResult
	LeaseToken      domain.LeaseToken
}

// ReportExecutionResult records a worker result.
func (u *ExecutionQueueUsecase) ReportExecutionResult(ctx context.Context, actor domain.Actor, command ReportExecutionResultCommand) error {
	if !actor.Role.HasCapability(domain.CapQueueReport) {
		return errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapQueueReport)
	}
	var retryExec *domain.Execution
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

		nextRunAt := e.NextRetryRunAt(time.Now())
		retryE, err := queues.CreateRetryUnsuccessful(ctx, e.Id, nextRunAt)
		if err != nil {
			return err
		}

		retryExec = &retryE

		return nil
	}); err != nil {
		return err
	}

	if retryExec != nil {
		if err := u.d.Trigger(ctx); err != nil {
			return err
		}
	}
	return nil
}

// NewExecutionQueueUsecase constructs a queue use case.
func NewExecutionQueueUsecase(s ExecutionQueueStore, d OutboxDispatcher) *ExecutionQueueUsecase {
	return &ExecutionQueueUsecase{s: s, d: d}
}
