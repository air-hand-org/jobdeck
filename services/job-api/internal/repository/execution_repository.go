package repository

import (
	"context"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	jobdb "jobdeck/services/job-api/internal/repository/gen"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
)

type executionQuerier interface {
	CreateExecutionForCurrentJob(ctx context.Context, arg jobdb.CreateExecutionForCurrentJobParams) (uuid.UUID, error)
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
	exec_id, err := r.q.CreateExecutionForCurrentJob(ctx, jobdb.CreateExecutionForCurrentJobParams{
		TriggerType: string(trigger),
		JobID:       job_id,
		OwnerID:     owner_id,
	})
	if err != nil {
		return domain.Execution{}, err
	}
	return r.FindById(ctx, owner_id, exec_id)
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
