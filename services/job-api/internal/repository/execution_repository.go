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

func (r *executionRepository) CreateForCurrentJob(ctx context.Context, ownerID domain.ActorId, jobID domain.JobId, trigger domain.ExecutionTrigger) (domain.Execution, error) {
	execID, err := r.q.CreateExecutionForCurrentJob(ctx, jobdb.CreateExecutionForCurrentJobParams{
		TriggerType: string(trigger),
		JobID:       jobID,
		OwnerID:     ownerID,
	})
	if err != nil {
		return domain.Execution{}, err
	}
	return r.FindById(ctx, ownerID, execID)
}

//nolint:revive // Existing internal API name is retained for compatibility.
func (r *executionRepository) FindById(ctx context.Context, ownerID domain.ActorId, executionID domain.ExecutionId) (domain.Execution, error) {
	exec, err := r.q.GetExecutionDetail(ctx, jobdb.GetExecutionDetailParams{
		ExecutionID: executionID,
		OwnerID:     ownerID,
	})
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

func (r *executionRepository) ListByJob(ctx context.Context, ownerID domain.ActorId, jobID domain.JobId, pageSize int32, pageToken string) (app.ListExecutionsResult, error) {
	var (
		execs []jobdb.Execution
		err   error
	)

	originalPageSize := pageSize
	pageSize++
	if originalPageSize <= 0 {
		return app.ListExecutionsResult{}, errors.Newf("page size must be positive : %v", originalPageSize)
	}

	if pageToken == "" {
		execs, err = r.q.ListExecutionsFirstPage(ctx, jobdb.ListExecutionsFirstPageParams{
			JobID:    jobID,
			OwnerID:  ownerID,
			PageSize: pageSize,
		})
	} else {
		token, err := decodeExecutionPageToken(pageToken)
		if err != nil {
			return app.ListExecutionsResult{}, err
		}
		createdAt, err := timeToPGType(token.CreatedAt)
		if err != nil {
			return app.ListExecutionsResult{}, err
		}
		var queryErr error
		execs, queryErr = r.q.ListExecutionsAfter(ctx, jobdb.ListExecutionsAfterParams{
			JobID:       jobID,
			OwnerID:     ownerID,
			CreatedAt:   createdAt,
			ExecutionID: token.ExecutionID,
			PageSize:    pageSize,
		})
		if queryErr != nil {
			return app.ListExecutionsResult{}, queryErr
		}
	}
	if err != nil {
		return app.ListExecutionsResult{}, err
	}

	hasNext := len(execs) > int(originalPageSize)
	if hasNext {
		execs = execs[:originalPageSize]
	}

	executions, err := lo.MapErr(execs, func(e jobdb.Execution, _ int) (domain.Execution, error) {
		return executionFromDB(e)
	})
	if err != nil {
		return app.ListExecutionsResult{}, err
	}
	var nextPageToken string
	if hasNext {
		last := executions[originalPageSize-1]
		nextPageToken, err = encodeExecutionPageToken(executionPageToken{
			CreatedAt:   last.CreatedAt,
			ExecutionID: last.Id,
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
