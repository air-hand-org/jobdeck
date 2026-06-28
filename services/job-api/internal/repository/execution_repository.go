package repository

import (
	"context"
	"encoding/json"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	jobdb "jobdeck/services/job-api/internal/repository/gen"
	"time"

	"github.com/samber/lo"
)

type executionQuerier interface {
	ClaimExecutions(ctx context.Context, arg jobdb.ClaimExecutionsParams) ([]jobdb.JobExecutionClaimDetail, error)
	CreateExecutionForCurrentJob(ctx context.Context, arg jobdb.CreateExecutionForCurrentJobParams) (jobdb.Execution, error)
	CreateRetryForUnsuccessfulExecution(ctx context.Context, arg jobdb.CreateRetryForUnsuccessfulExecutionParams) (jobdb.Execution, error)
	ReportExecutionResult(ctx context.Context, arg jobdb.ReportExecutionResultParams) (jobdb.Execution, error)
}

var _ executionQuerier = (*jobdb.Queries)(nil)

type executionRepository struct {
	q executionQuerier
}

var _ app.ExecutionRepository = (*executionRepository)(nil)

func (r *executionRepository) CreateForCurrentJob(ctx context.Context, owner_id domain.UserId, job_id domain.JobId, trigger domain.ExecutionTrigger) (domain.Execution, error) {
	exec, err := r.q.CreateExecutionForCurrentJob(ctx, jobdb.CreateExecutionForCurrentJobParams{
		TriggerType: string(trigger),
		JobID:       job_id,
		OwnerID:     owner_id,
	})
	if err != nil {
		return domain.Execution{}, err
	}
	return executionFromDB(exec)
}

func (r *executionRepository) CreateRetryUnsuccessful(ctx context.Context, execution_id domain.ExecutionId, next_run_at time.Time) (domain.Execution, error) {
	n, err := timeToPGType(next_run_at)
	if err != nil {
		return domain.Execution{}, err
	}
	exec, err := r.q.CreateRetryForUnsuccessfulExecution(ctx, jobdb.CreateRetryForUnsuccessfulExecutionParams{
		ExecutionID: execution_id,
		NextRunAt:   n,
	})
	if err != nil {
		return domain.Execution{}, err
	}
	return executionFromDB(exec)
}

func (r *executionRepository) Claims(ctx context.Context, max_count int32) ([]domain.ClaimedExecution, error) {
	// TODO
	lease_expires_at, err := timeToPGType(time.Now().Add(5 * time.Minute))
	if err != nil {
		return []domain.ClaimedExecution{}, err
	}
	claims, err := r.q.ClaimExecutions(ctx, jobdb.ClaimExecutionsParams{
		MaxSize:        max_count,
		LeaseExpiresAt: lease_expires_at,
	})
	if err != nil {
		return []domain.ClaimedExecution{}, err
	}
	return lo.MapErr(claims, func(e jobdb.JobExecutionClaimDetail, _ int) (domain.ClaimedExecution, error) {
		return claimExecutionFromDB(e)
	})
}

func (r *executionRepository) ReportResult(ctx context.Context, execution_id domain.ExecutionId, status domain.ExecutionStatus, lease_token domain.LeaseToken, result domain.ExecutionResult) (domain.Execution, error) {
	res, err := json.Marshal(result.Result)
	if err != nil {
		return domain.Execution{}, err
	}
	exec, err := r.q.ReportExecutionResult(ctx, jobdb.ReportExecutionResultParams{
		ExecutionID:  execution_id,
		LeaseToken:   &lease_token,
		Status:       string(status),
		ErrorMessage: textToPGType(result.ErrorMessage),
		Result:       res,
	})
	if err != nil {
		return domain.Execution{}, err
	}
	return executionFromDB(exec)
}

func newExecutionRepository(q executionQuerier) *executionRepository {
	return &executionRepository{q: q}
}
