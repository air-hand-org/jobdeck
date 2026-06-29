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

type executionQuerier interface {
	ClaimExecutions(ctx context.Context, arg jobdb.ClaimExecutionsParams) ([]jobdb.JobExecutionClaimDetail, error)
	CreateExecutionForCurrentJob(ctx context.Context, arg jobdb.CreateExecutionForCurrentJobParams) (jobdb.Execution, error)
	CreateRetryForUnsuccessfulExecution(ctx context.Context, arg jobdb.CreateRetryForUnsuccessfulExecutionParams) (jobdb.Execution, error)
	ReportExecutionResult(ctx context.Context, arg jobdb.ReportExecutionResultParams) (jobdb.Execution, error)

	GetExecutionDetail(ctx context.Context, arg jobdb.GetExecutionDetailParams) (jobdb.Execution, error)
	ListExecutionEvents(ctx context.Context, executionID uuid.UUID) ([]jobdb.JobExecutionEvent, error)
	ListExecutionsAfter(ctx context.Context, arg jobdb.ListExecutionsAfterParams) ([]jobdb.Execution, error)
	ListExecutionsFirstPage(ctx context.Context, arg jobdb.ListExecutionsFirstPageParams) ([]jobdb.Execution, error)
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

func (r *executionRepository) FindById(ctx context.Context, owner_id domain.UserId, execution_id domain.ExecutionId) (domain.Execution, error) {
	exec, err := r.q.GetExecutionDetail(ctx, jobdb.GetExecutionDetailParams{
		ExecutionID: execution_id,
		OwnerID:     owner_id,
	})
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

func (r *executionRepository) ListByJob(ctx context.Context, owner_id domain.UserId, job_id domain.JobId, page_size int32, page_token string) (app.ListExecutionsResult, error) {
	var (
		execs []jobdb.Execution
		err   error
	)

	org_page_size := page_size
	page_size += 1
	if org_page_size <= 0 {
		return app.ListExecutionsResult{}, errors.Newf("page size must be positive : %v", org_page_size)
	}

	if page_token == "" {
		execs, err = r.q.ListExecutionsFirstPage(ctx, jobdb.ListExecutionsFirstPageParams{
			JobID:    job_id,
			OwnerID:  owner_id,
			PageSize: page_size,
		})
	} else {
		token, err := decodeExecutionPageToken(page_token)
		if err != nil {
			return app.ListExecutionsResult{}, err
		}
		c_at, err := timeToPGType(token.CreatedAt)
		if err != nil {
			return app.ListExecutionsResult{}, err
		}
		execs, err = r.q.ListExecutionsAfter(ctx, jobdb.ListExecutionsAfterParams{
			JobID:       job_id,
			OwnerID:     owner_id,
			CreatedAt:   c_at,
			ExecutionID: token.ExecutionId,
			PageSize:    page_size,
		})
	}
	if err != nil {
		return app.ListExecutionsResult{}, err
	}

	hasNext := len(execs) > int(org_page_size)
	if hasNext {
		execs = execs[:org_page_size]
	}

	executions, err := lo.MapErr(execs, func(e jobdb.Execution, _ int) (domain.Execution, error) {
		return executionFromDB(e)
	})
	if err != nil {
		return app.ListExecutionsResult{}, err
	}
	var nextPageToken string
	if hasNext {
		last := executions[org_page_size-1]
		nextPageToken, err = encodeExecutionPageToken(executionPageToken{
			CreatedAt:   last.CreatedAt,
			ExecutionId: last.Id,
		})
		if err != nil {
			return app.ListExecutionsResult{}, err
		}
	}

	return app.ListExecutionsResult{
		Executions:    executions,
		NextPageToken: nextPageToken,
	}, nil
}

func newExecutionRepository(q executionQuerier) *executionRepository {
	return &executionRepository{q: q}
}
