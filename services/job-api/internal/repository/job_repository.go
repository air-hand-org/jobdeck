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

func (r *jobRepository) Create(ctx context.Context, owner_id domain.ActorId, name string, spec domain.JobSpec) (domain.Job, error) {
	cfg, err := json.Marshal(spec.Config)
	if err != nil {
		return domain.Job{}, err
	}
	job_id, err := r.q.CreateJob(ctx, jobdb.CreateJobParams{
		OwnerID:             owner_id,
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

	return r.FindById(ctx, owner_id, job_id)
}

func (r *jobRepository) UpdateSpec(ctx context.Context, owner_id domain.ActorId, job_id domain.JobId, spec domain.JobSpec) (domain.Job, error) {
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
		JobID:               job_id,
		OwnerID:             owner_id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Job{}, errors.Wrapf(domain.ErrJobNotFound, "job %v was not found", job_id)
		}
		return domain.Job{}, err
	}
	return r.FindById(ctx, owner_id, job_id)
}

func (r *jobRepository) Rename(ctx context.Context, owner_id domain.ActorId, job_id domain.JobId, name string) error {
	cnt, err := r.q.RenameJob(ctx, jobdb.RenameJobParams{
		Name:    name,
		JobID:   job_id,
		OwnerID: owner_id,
	})
	if err != nil {
		return err
	}
	if cnt != 1 {
		return domain.ErrJobNotFound
	}
	return nil
}

func (r *jobRepository) SetEnabled(ctx context.Context, owner_id domain.ActorId, job_id domain.JobId, enabled bool) (domain.Job, error) {
	_, err := r.q.SetJobEnabled(ctx, jobdb.SetJobEnabledParams{
		Enabled: enabled,
		JobID:   job_id,
		OwnerID: owner_id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Job{}, errors.Wrapf(domain.ErrJobNotFound, "job %v was not found", job_id)
		}
		return domain.Job{}, err
	}
	return r.FindById(ctx, owner_id, job_id)
}

func (r *jobRepository) Delete(ctx context.Context, owner_id domain.ActorId, job_id domain.JobId) error {
	affected, err := r.q.DeleteJob(ctx, jobdb.DeleteJobParams{
		JobID:   job_id,
		OwnerID: owner_id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.Wrapf(domain.ErrJobNotFound, "job %v was not found", job_id)
		}
		return err
	}
	if affected != 1 {
		return errors.Newf("job %v was deleted, but affected: %d", job_id, affected)
	}
	return nil
}

func (r *jobRepository) FindById(ctx context.Context, owner_id domain.ActorId, job_id domain.JobId) (domain.Job, error) {
	j, err := r.q.GetJob(ctx, jobdb.GetJobParams{
		JobID:   job_id,
		OwnerID: owner_id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Job{}, errors.Wrapf(domain.ErrJobNotFound, "job %v was not found", job_id)
		}
		return domain.Job{}, err
	}
	return jobFromDB(j)
}

func (r *jobRepository) ListByOwner(ctx context.Context, owner_id domain.ActorId, page_size int32, page_token string) (app.ListJobsResult, error) {
	var (
		jobs []jobdb.Job
		err  error
	)

	// for paging
	org_page_size := page_size
	page_size += 1
	if org_page_size <= 0 {
		return app.ListJobsResult{}, errors.New("page size must be positive")
	}

	if page_token == "" {
		jobs, err = r.q.ListJobsFirstPage(ctx, jobdb.ListJobsFirstPageParams{
			OwnerID:  owner_id,
			PageSize: page_size,
		})
	} else {
		token, err := decodeJobPageToken(page_token)
		if err != nil {
			return app.ListJobsResult{}, err
		}
		c_at, err := timeToPGType(token.CreatedAt)
		if err != nil {
			return app.ListJobsResult{}, err
		}
		jobs, err = r.q.ListJobsAfter(ctx, jobdb.ListJobsAfterParams{
			OwnerID:   owner_id,
			CreatedAt: c_at,
			JobID:     token.JobId,
			PageSize:  page_size,
		})
	}
	if err != nil {
		return app.ListJobsResult{}, err
	}

	hasNext := len(jobs) > int(org_page_size)
	if hasNext {
		jobs = jobs[:org_page_size]
	}

	js, err := lo.MapErr(jobs, func(j jobdb.Job, _ int) (domain.Job, error) {
		return jobFromDB(j)
	})
	if err != nil {
		return app.ListJobsResult{}, err
	}

	var nextPageToken string
	if hasNext {
		last := js[org_page_size-1]
		nextPageToken, err = encodeJobPageToken(jobPageToken{
			CreatedAt: last.CreatedAt,
			JobId:     last.Id,
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
