package repository

import (
	"context"
	"encoding/json"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	jobdb "jobdeck/services/job-api/internal/repository/gen"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
)

type executionQueueQuerier interface {
	ClaimExecutions(ctx context.Context, arg jobdb.ClaimExecutionsParams) ([]uuid.UUID, error)
	CreateRetryForUnsuccessfulExecution(ctx context.Context, arg jobdb.CreateRetryForUnsuccessfulExecutionParams) (uuid.UUID, error)
	ReportExecutionResult(ctx context.Context, arg jobdb.ReportExecutionResultParams) (uuid.UUID, error)

	GetExecutionDetailDirect(ctx context.Context, executionID uuid.UUID) (jobdb.Execution, error)
	ListExecutionEvents(ctx context.Context, executionID uuid.UUID) ([]jobdb.JobExecutionEvent, error)
	ListExecutionClaimDetails(ctx context.Context, executionIdList []uuid.UUID) ([]jobdb.JobExecutionClaimDetail, error)
}

var _ executionQueueQuerier = (*jobdb.Queries)(nil)

type executionQueueRepository struct {
	q executionQueueQuerier
}

var _ app.ExecutionQueueRepository = (*executionQueueRepository)(nil)

func (r *executionQueueRepository) CreateRetryUnsuccessful(ctx context.Context, execution_id domain.ExecutionId, next_run_at time.Time) (domain.Execution, error) {
	n, err := timeToPGType(next_run_at)
	if err != nil {
		return domain.Execution{}, err
	}
	exec_id, err := r.q.CreateRetryForUnsuccessfulExecution(ctx, jobdb.CreateRetryForUnsuccessfulExecutionParams{
		ExecutionID: execution_id,
		NextRunAt:   n,
	})
	if err != nil {
		return domain.Execution{}, err
	}
	return r.findById(ctx, exec_id)
}

func (r *executionQueueRepository) Claims(ctx context.Context, max_count int32) ([]domain.ClaimedExecution, error) {
	// TODO
	lease_expires_at, err := timeToPGType(time.Now().Add(5 * time.Minute))
	if err != nil {
		return []domain.ClaimedExecution{}, err
	}
	claim_exec_id_list, err := r.q.ClaimExecutions(ctx, jobdb.ClaimExecutionsParams{
		MaxSize:        max_count,
		LeaseExpiresAt: lease_expires_at,
	})
	if err != nil {
		return []domain.ClaimedExecution{}, errors.Wrap(err, "failed to claim executions")
	}
	claims, err := r.q.ListExecutionClaimDetails(ctx, claim_exec_id_list)
	if err != nil {
		return []domain.ClaimedExecution{}, errors.Wrap(err, "failed to list execution claim details")
	}
	return lo.MapErr(claims, func(e jobdb.JobExecutionClaimDetail, _ int) (domain.ClaimedExecution, error) {
		return claimExecutionFromDB(e)
	})
}

func (r *executionQueueRepository) GetReportTarget(ctx context.Context, execution_id domain.ExecutionId, lease_token domain.LeaseToken) (domain.Execution, error) {
	exec, err := r.findById(ctx, execution_id)
	if err != nil {
		return domain.Execution{}, err
	}

	if exec.Status != domain.EXECUTION_STATUS_RUNNING || exec.LeaseToken != lease_token {
		return domain.Execution{}, domain.ErrExecutionNotReportable
	}

	return exec, nil
}

func (r *executionQueueRepository) ReportResult(ctx context.Context, execution_id domain.ExecutionId, status domain.ExecutionStatus, lease_token domain.LeaseToken, result domain.ExecutionResult) (domain.Execution, error) {
	res, err := json.Marshal(result.Result)
	if err != nil {
		return domain.Execution{}, err
	}
	_, err = r.q.ReportExecutionResult(ctx, jobdb.ReportExecutionResultParams{
		ExecutionID: execution_id,
		LeaseToken:  &lease_token,
		Status:      string(status),
		Result:      res,
	})
	if err != nil {
		return domain.Execution{}, err
	}
	return r.findById(ctx, execution_id)
}

// TODO: duplicate ExecutionRepository.FindById
func (r *executionQueueRepository) findById(ctx context.Context, execution_id domain.ExecutionId) (domain.Execution, error) {
	exec, err := r.q.GetExecutionDetailDirect(ctx, execution_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Execution{}, errors.Wrapf(domain.ErrExecutionNotFound, "execution %v was not found", execution_id)
		}
		return domain.Execution{}, err
	}
	events, err := r.q.ListExecutionEvents(ctx, exec.ExecutionID)
	if err != nil {
		return domain.Execution{}, err
	}
	execution, err := executionFromDB(exec)
	if err != nil {
		return domain.Execution{}, err
	}
	execution_events, err := executionEventsFromDB(events)
	if err != nil {
		return domain.Execution{}, err
	}
	execution.Events = execution_events
	return execution, nil
}

func newExecutionQueueRepository(q executionQueueQuerier) *executionQueueRepository {
	return &executionQueueRepository{q: q}
}
