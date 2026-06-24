package app

import (
	"context"
	"fmt"
	"jobdeck/services/job-api/internal/domain"
	"time"
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

func (u *ExecutionUsecase) ClaimExecutions(ctx context.Context, max_count int32) ([]domain.ClaimedExecution, error) {
	if max_count <= 0 {
		return nil, fmt.Errorf("max_count must be positive: %v", max_count)
	}
	var executions []domain.ClaimedExecution
	err := u.s.WithinExecutionTx(ctx, func(ctx context.Context, execs ExecutionRepository) error {
		res, err := execs.Claims(ctx, max_count)
		if err != nil {
			return err
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

func (u *ExecutionUsecase) ReportExecutionResult(ctx context.Context, command ReportExecutionResultCommand) error {
	return u.s.WithinExecutionTx(ctx, func(ctx context.Context, execs ExecutionRepository) error {
		e, err := execs.ReportResult(ctx, command.ExecutionId, command.ExecutionStatus, command.LeaseToken, command.ExecutionResult)
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

		_, err = execs.CreateRetryUnsuccessful(ctx, e.Id, next_run_at)
		if err != nil {
			return err
		}
		return nil
	})
}

func NewExecutionUsecase(s ExecutionStore) *ExecutionUsecase {
	return &ExecutionUsecase{s: s}
}
