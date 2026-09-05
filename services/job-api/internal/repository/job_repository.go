package repository

import (
	"context"
	"encoding/json"

	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	jobdb "jobdeck/services/job-api/internal/repository/gen"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
)

type jobQuerier interface {
	CreateJob(ctx context.Context, arg jobdb.CreateJobParams) (uuid.UUID, error)
	DeleteJob(ctx context.Context, arg jobdb.DeleteJobParams) (int64, error)
	GetJob(ctx context.Context, arg jobdb.GetJobParams) (jobdb.Job, error)
	ListJobsAfter(ctx context.Context, arg jobdb.ListJobsAfterParams) ([]jobdb.Job, error)
	ListJobsFirstPage(ctx context.Context, arg jobdb.ListJobsFirstPageParams) ([]jobdb.Job, error)
	RenameJob(ctx context.Context, arg jobdb.RenameJobParams) (int64, error)
	SetJobEnabled(ctx context.Context, arg jobdb.SetJobEnabledParams) (uuid.UUID, error)
	UpdateJobSpec(ctx context.Context, arg jobdb.UpdateJobSpecParams) (uuid.UUID, error)
}

var _ jobQuerier = (*jobdb.Queries)(nil)

type jobRepository struct {
	q jobQuerier
}

func newJobRepository(q jobQuerier) *jobRepository {
	return &jobRepository{q: q}
}

var _ app.JobRepository = (*jobRepository)(nil)

func (r *jobRepository) Create(ctx context.Context, ownerID domain.ActorId, name string, spec domain.JobSpec) (domain.Job, error) {
	cfg, err := json.Marshal(spec.Config)
	if err != nil {
		return domain.Job{}, err
	}
	jobID, err := r.q.CreateJob(ctx, jobdb.CreateJobParams{
		OwnerID:             ownerID,
		Name:                name,
		Type:                string(spec.Type),
		Config:              cfg,
		RetryMaxAttempts:    spec.RetryPolicy.MaxAttempts,
		RetryBackoffType:    string(spec.RetryPolicy.BackoffType),
		RetryInitialDelayMs: spec.RetryPolicy.InitialDelay.Milliseconds(),
		RetryMaxDelayMs:     spec.RetryPolicy.MaxDelay.Milliseconds(),
	})
	if err != nil {
		return domain.Job{}, err
	}

	return r.FindByID(ctx, ownerID, jobID)
}

func (r *jobRepository) UpdateSpec(ctx context.Context, ownerID domain.ActorId, jobID domain.JobId, spec domain.JobSpec) (domain.Job, error) {
	cfg, err := json.Marshal(spec.Config)
	if err != nil {
		return domain.Job{}, err
	}
	_, err = r.q.UpdateJobSpec(ctx, jobdb.UpdateJobSpecParams{
		Type:   string(spec.Type),
		Config: cfg,

		RetryMaxAttempts:    spec.RetryPolicy.MaxAttempts,
		RetryBackoffType:    string(spec.RetryPolicy.BackoffType),
		RetryInitialDelayMs: spec.RetryPolicy.InitialDelay.Milliseconds(),
		RetryMaxDelayMs:     spec.RetryPolicy.MaxDelay.Milliseconds(),
		JobID:               jobID,
		OwnerID:             ownerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Job{}, errors.Wrapf(domain.ErrJobNotFound, "job %v was not found", jobID)
		}
		return domain.Job{}, err
	}
	return r.FindByID(ctx, ownerID, jobID)
}

func (r *jobRepository) Rename(ctx context.Context, ownerID domain.ActorId, jobID domain.JobId, name string) error {
	cnt, err := r.q.RenameJob(ctx, jobdb.RenameJobParams{
		Name:    name,
		JobID:   jobID,
		OwnerID: ownerID,
	})
	if err != nil {
		return err
	}
	if cnt != 1 {
		return domain.ErrJobNotFound
	}
	return nil
}

func (r *jobRepository) SetEnabled(ctx context.Context, ownerID domain.ActorId, jobID domain.JobId, enabled bool) (domain.Job, error) {
	_, err := r.q.SetJobEnabled(ctx, jobdb.SetJobEnabledParams{
		Enabled: enabled,
		JobID:   jobID,
		OwnerID: ownerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Job{}, errors.Wrapf(domain.ErrJobNotFound, "job %v was not found", jobID)
		}
		return domain.Job{}, err
	}
	return r.FindByID(ctx, ownerID, jobID)
}

func (r *jobRepository) Delete(ctx context.Context, ownerID domain.ActorId, jobID domain.JobId) error {
	affected, err := r.q.DeleteJob(ctx, jobdb.DeleteJobParams{
		JobID:   jobID,
		OwnerID: ownerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.Wrapf(domain.ErrJobNotFound, "job %v was not found", jobID)
		}
		return err
	}
	if affected != 1 {
		return errors.Newf("job %v was deleted, but affected: %d", jobID, affected)
	}
	return nil
}

func (r *jobRepository) FindByID(ctx context.Context, ownerID domain.ActorId, jobID domain.JobId) (domain.Job, error) {
	j, err := r.q.GetJob(ctx, jobdb.GetJobParams{
		JobID:   jobID,
		OwnerID: ownerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Job{}, errors.Wrapf(domain.ErrJobNotFound, "job %v was not found", jobID)
		}
		return domain.Job{}, err
	}
	return jobFromDB(j)
}

func (r *jobRepository) ListByOwner(ctx context.Context, ownerID domain.ActorId, pageSize int32, pageToken string) (app.ListJobsResult, error) {
	var (
		jobs []jobdb.Job
		err  error
	)

	// for paging
	orgPageSize := pageSize
	pageSize++
	if orgPageSize <= 0 {
		return app.ListJobsResult{}, errors.New("page size must be positive")
	}

	if pageToken == "" {
		jobs, err = r.q.ListJobsFirstPage(ctx, jobdb.ListJobsFirstPageParams{
			OwnerID:  ownerID,
			PageSize: pageSize,
		})
	} else {
		token, err := decodeJobPageToken(pageToken)
		if err != nil {
			return app.ListJobsResult{}, err
		}
		cAt, err := timeToPGType(token.CreatedAt)
		if err != nil {
			return app.ListJobsResult{}, err
		}
		var queryErr error
		jobs, queryErr = r.q.ListJobsAfter(ctx, jobdb.ListJobsAfterParams{
			OwnerID:   ownerID,
			CreatedAt: cAt,
			JobID:     token.JobID,
			PageSize:  pageSize,
		})
		if queryErr != nil {
			return app.ListJobsResult{}, queryErr
		}
	}
	if err != nil {
		return app.ListJobsResult{}, err
	}

	hasNext := len(jobs) > int(orgPageSize)
	if hasNext {
		jobs = jobs[:orgPageSize]
	}

	js, err := lo.MapErr(jobs, func(j jobdb.Job, _ int) (domain.Job, error) {
		return jobFromDB(j)
	})
	if err != nil {
		return app.ListJobsResult{}, err
	}

	var nextPageToken string
	if hasNext {
		last := js[orgPageSize-1]
		nextPageToken, err = encodeJobPageToken(jobPageToken{
			CreatedAt: last.CreatedAt,
			JobID:     last.Id,
		})
		if err != nil {
			return app.ListJobsResult{}, err
		}
	}

	return app.ListJobsResult{
		Jobs:          js,
		NextPageToken: nextPageToken,
	}, nil
}
