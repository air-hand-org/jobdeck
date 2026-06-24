package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
)

//go:generate moq -out gen_job_store_moq_test.go . JobStore
type JobStore interface {
	Jobs() JobRepository
	WithinJobTx(ctx context.Context, fn func(ctx context.Context, jobs JobRepository) error) error
}

type JobUsecase struct {
	s JobStore
}

func (u *JobUsecase) CreateJob(ctx context.Context, owner_id domain.UserId, name string, spec domain.JobSpec) (domain.Job, error) {
	var job domain.Job
	err := u.s.WithinJobTx(ctx, func(ctx context.Context, jobs JobRepository) error {
		j, err := jobs.Create(ctx, owner_id, name, spec)
		if err != nil {
			return err
		}
		job = j
		return nil
	})
	return job, err
}

func (u *JobUsecase) UpdateJobMetadata(ctx context.Context, owner_id domain.UserId, job_id domain.JobId, name string, spec domain.JobSpec) (domain.Job, error) {
	var job domain.Job
	err := u.s.WithinJobTx(ctx, func(ctx context.Context, jobs JobRepository) error {
		err := jobs.Rename(ctx, owner_id, job_id, name)
		if err != nil {
			return err
		}
		j, err := jobs.UpdateSpec(ctx, owner_id, job_id, spec)
		if err != nil {
			return err
		}
		job = j
		return nil
	})
	return job, err
}

func (u *JobUsecase) SetJobEnabled(ctx context.Context, owner_id domain.UserId, job_id domain.JobId, enabled bool) (domain.Job, error) {
	var job domain.Job
	err := u.s.WithinJobTx(ctx, func(ctx context.Context, jobs JobRepository) error {
		j, err := jobs.SetEnabled(ctx, owner_id, job_id, enabled)
		if err != nil {
			return err
		}
		job = j
		return nil
	})
	return job, err
}

func (u *JobUsecase) DeleteJob(ctx context.Context, owner_id domain.UserId, job_id domain.JobId) error {
	return u.s.WithinJobTx(ctx, func(ctx context.Context, jobs JobRepository) error {
		return jobs.Delete(ctx, owner_id, job_id)
	})
}

func (u *JobUsecase) GetJob(ctx context.Context, owner_id domain.UserId, job_id domain.JobId) (domain.Job, error) {
	return u.s.Jobs().FindById(ctx, owner_id, job_id)
}

func (u *JobUsecase) ListJobs(ctx context.Context, owner_id domain.UserId, page_size int32, page_token string) (ListJobsResult, error) {
	return u.s.Jobs().ListByOwner(ctx, owner_id, page_size, page_token)
}

func NewJobUsecase(s JobStore) *JobUsecase {
	return &JobUsecase{s: s}
}
