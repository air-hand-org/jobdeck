package app

import (
	"context"
	"fmt"
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
}

func (u *ExecutionQueueUsecase) ClaimExecutions(ctx context.Context, max_count int32) ([]domain.ClaimedExecution, error) {
	if max_count <= 0 {
		return nil, fmt.Errorf("max_count must be positive: %v", max_count)
	}
	var executions []domain.ClaimedExecution
	err := u.s.WithinExecutionQueueTx(ctx, func(ctx context.Context, queues ExecutionQueueRepository) error {
		res, err := queues.Claims(ctx, max_count)
		if err != nil {
			return errors.Wrapf(err, "failed to claims %d", max_count)
		}
		executions = res
		return nil
	})
	if err != nil {
		return []domain.ClaimedExecution{}, err
	}
	return executions, nil
}

type ReportExecutionResultCommand struct {
	ExecutionId     domain.ExecutionId
	ExecutionStatus domain.ExecutionStatus
	ExecutionResult domain.ExecutionResult
	LeaseToken      domain.LeaseToken
}

func (u *ExecutionQueueUsecase) ReportExecutionResult(ctx context.Context, command ReportExecutionResultCommand) error {
	return u.s.WithinExecutionQueueTx(ctx, func(ctx context.Context, queues ExecutionQueueRepository) error {
		e, err := queues.ReportResult(ctx, command.ExecutionId, command.ExecutionStatus, command.LeaseToken, command.ExecutionResult)
		if err != nil {
			return err
		}

		// TODO: Retry
		retry := false
		if !retry {
			return nil
		}
		// FIXME
		next_run_at := time.Now().Add(10 * time.Second)

		_, err = queues.CreateRetryUnsuccessful(ctx, e.Id, next_run_at)
		if err != nil {
			return err
		}
		return nil
	})
}

func NewExecutionQueueUsecase(s ExecutionQueueStore) *ExecutionQueueUsecase {
	return &ExecutionQueueUsecase{s: s}
}
