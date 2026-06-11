package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
)

type JobUsecase struct {
	// TODO: transaction control
	repo JobRepository
}

func (u *JobUsecase) CreateJob(ctx context.Context, owner_id domain.UserId, spec domain.JobSpec) (domain.Job, error) {
	return u.repo.Create(ctx, owner_id, spec)
}

func (u *JobUsecase) UpdateJob(ctx context.Context, owner_id domain.UserId, job_id domain.JobId, enabled bool, spec domain.JobSpec) (domain.Job, error) {
	job, err := u.repo.FindById(ctx, owner_id, job_id)
	if err != nil {
		return domain.Job{}, err
	}
	job.Enabled = enabled
	job.Spec = spec
	return u.repo.Update(ctx, owner_id, job)
}

func (u *JobUsecase) DeleteJob(ctx context.Context, owner_id domain.UserId, job_id domain.JobId) error {
	return u.repo.Delete(ctx, owner_id, job_id)
}

func (u *JobUsecase) GetJob(ctx context.Context, owner_id domain.UserId, job_id domain.JobId) (domain.Job, error) {
	return u.repo.FindById(ctx, owner_id, job_id)
}

func (u *JobUsecase) ListJobs(ctx context.Context, owner_id domain.UserId) ([]domain.Job, error) {
	return u.repo.ListByOwner(ctx, owner_id)
}

func NewJobUsecase(repo JobRepository) *JobUsecase {
	return &JobUsecase{repo: repo}
}
