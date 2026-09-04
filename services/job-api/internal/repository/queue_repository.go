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
)

type executionQueueQuerier interface {
	ClaimExecution(ctx context.Context, arg jobdb.ClaimExecutionParams) (uuid.UUID, error)
	CreateRetryForUnsuccessfulExecution(ctx context.Context, arg jobdb.CreateRetryForUnsuccessfulExecutionParams) (uuid.UUID, error)
	ReportExecutionResult(ctx context.Context, arg jobdb.ReportExecutionResultParams) (uuid.UUID, error)

	GetExecutionClaimDetail(ctx context.Context, executionID uuid.UUID) (jobdb.JobExecutionClaimDetail, error)
	GetExecutionDetailDirect(ctx context.Context, executionID uuid.UUID) (jobdb.Execution, error)
	ListExecutionEvents(ctx context.Context, executionID uuid.UUID) ([]jobdb.JobExecutionEvent, error)
	ListExecutionClaimDetails(ctx context.Context, executionIDList []uuid.UUID) ([]jobdb.JobExecutionClaimDetail, error)
}

var _ executionQueueQuerier = (*jobdb.Queries)(nil)

type executionQueueRepository struct {
	q executionQueueQuerier
}

var _ app.ExecutionQueueRepository = (*executionQueueRepository)(nil)

func (r *executionQueueRepository) CreateRetryUnsuccessful(ctx context.Context, executionID domain.ExecutionId, nextRunAt time.Time) (domain.Execution, error) {
	n, err := timeToPGType(nextRunAt)
	if err != nil {
		return domain.Execution{}, err
	}
	execID, err := r.q.CreateRetryForUnsuccessfulExecution(ctx, jobdb.CreateRetryForUnsuccessfulExecutionParams{
		ExecutionID: executionID,
		NextRunAt:   n,
	})
	if err != nil {
		return domain.Execution{}, err
	}
	return r.findByID(ctx, execID)
}

func (r *executionQueueRepository) Claim(ctx context.Context, executionID domain.ExecutionId) (domain.ClaimedExecution, error) {
	// TODO
	leaseExpiresAt, err := timeToPGType(time.Now().Add(5 * time.Minute))
	if err != nil {
		return domain.ClaimedExecution{}, err
	}
	claimExecID, err := r.q.ClaimExecution(ctx, jobdb.ClaimExecutionParams{
		ExecutionID:    executionID,
		LeaseExpiresAt: leaseExpiresAt,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ClaimedExecution{}, errors.Wrapf(domain.ErrExecutionHasAlreadyConsumed, "failed to claim execution: %v", err)
		}
		return domain.ClaimedExecution{}, errors.Wrap(err, "failed to claim execution")
	}
	claim, err := r.q.GetExecutionClaimDetail(ctx, claimExecID)
	if err != nil {
		return domain.ClaimedExecution{}, errors.Wrapf(domain.ErrExecutionNotFound, "failed to get execution claim detail: %v", err)
	}
	return claimExecutionFromDB(claim)
}

func (r *executionQueueRepository) GetReportTarget(ctx context.Context, executionID domain.ExecutionId, leaseToken domain.LeaseToken) (domain.Execution, error) {
	exec, err := r.findByID(ctx, executionID)
	if err != nil {
		return domain.Execution{}, err
	}

	if exec.Status != domain.EXECUTION_STATUS_RUNNING || exec.LeaseToken != leaseToken {
		return domain.Execution{}, domain.ErrExecutionNotReportable
	}

	return exec, nil
}

func (r *executionQueueRepository) ReportResult(ctx context.Context, executionID domain.ExecutionId, status domain.ExecutionStatus, leaseToken domain.LeaseToken, result domain.ExecutionResult) (domain.Execution, error) {
	res, err := json.Marshal(result.Result)
	if err != nil {
		return domain.Execution{}, err
	}
	_, err = r.q.ReportExecutionResult(ctx, jobdb.ReportExecutionResultParams{
		ExecutionID: executionID,
		LeaseToken:  &leaseToken,
		Status:      string(status),
		Result:      res,
	})
	if err != nil {
		return domain.Execution{}, err
	}
	return r.findByID(ctx, executionID)
}

// TODO: duplicate ExecutionRepository.FindById
func (r *executionQueueRepository) findByID(ctx context.Context, executionID domain.ExecutionId) (domain.Execution, error) {
	exec, err := r.q.GetExecutionDetailDirect(ctx, executionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Execution{}, errors.Wrapf(domain.ErrExecutionNotFound, "execution %v was not found", executionID)
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
	executionEvents, err := executionEventsFromDB(events)
	if err != nil {
		return domain.Execution{}, err
	}
	execution.Events = executionEvents
	return execution, nil
}

func newExecutionQueueRepository(q executionQueueQuerier) *executionQueueRepository {
	return &executionQueueRepository{q: q}
}
