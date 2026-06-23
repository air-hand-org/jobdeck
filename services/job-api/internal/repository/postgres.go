package repository

import (
	"context"
	"encoding/json"

	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	jobdb "jobdeck/services/job-api/internal/repository/gen"

	"github.com/cockroachdb/errors"
	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
)

type jobRepository struct {
	q jobdb.Querier
}

func NewJobRepository(q jobdb.Querier) app.JobRepository {
	return &jobRepository{q: q}
}

var _ app.JobRepository = &jobRepository{}

func (r *jobRepository) Create(ctx context.Context, owner_id domain.UserId, spec domain.JobSpec) (domain.Job, error) {
	cfg, err := json.Marshal(spec.Config)
	if err != nil {
		return domain.Job{}, err
	}
	job, err := r.q.CreateJob(ctx, jobdb.CreateJobParams{
		OwnerID:             owner_id,
		Name:                spec.Name,
		Type:                string(spec.Type),
		Config:              cfg,
		RetryMaxAttempts:    spec.RetryPolicy.MaxAttempts,
		RetryBackoffType:    string(spec.RetryPolicy.BackoffType),
		RetryInitialDelayMs: spec.RetryPolicy.InitialDelay.Milliseconds(),
		RetryMaxDelayMs:     spec.RetryPolicy.InitialDelay.Milliseconds(),
	})

	if err != nil {
		return domain.Job{}, err
	}

	return jobFromDB(job)
}

func (r *jobRepository) Update(ctx context.Context, owner_id domain.UserId, job domain.Job) (domain.Job, error) {
	cfg, err := json.Marshal(job.Spec.Config)
	if err != nil {
		return domain.Job{}, err
	}
	j, err := r.q.UpdateJob(ctx, jobdb.UpdateJobParams{
		Name:    job.Spec.Name,
		Enabled: job.Enabled,
		Type:    string(job.Spec.Type),
		Config:  cfg,

		RetryMaxAttempts:    job.Spec.RetryPolicy.MaxAttempts,
		RetryBackoffType:    string(job.Spec.RetryPolicy.BackoffType),
		RetryInitialDelayMs: job.Spec.RetryPolicy.InitialDelay.Milliseconds(),
		RetryMaxDelayMs:     job.Spec.RetryPolicy.MaxDelay.Milliseconds(),
		JobID:               job.Id,
		OwnerID:             job.OwnerId,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Job{}, errors.Wrapf(domain.ErrJobNotFound, "job %v was not found", job.Id)
		}
		return domain.Job{}, err
	}
	return jobFromDB(j)
}

func (r *jobRepository) Delete(ctx context.Context, owner_id domain.UserId, job_id domain.JobId) error {
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

func (r *jobRepository) FindById(ctx context.Context, owner_id domain.UserId, job_id domain.JobId) (domain.Job, error) {
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

func (r *jobRepository) ListByOwner(ctx context.Context, owner_id domain.UserId) ([]domain.Job, error) {
	jobs, err := r.q.ListJobsByOwner(ctx, owner_id)

	if err != nil {
		return nil, err
	}

	return lo.MapErr(jobs, func(j jobdb.Job, _ int) (domain.Job, error) {
		return jobFromDB(j)
	})
}
