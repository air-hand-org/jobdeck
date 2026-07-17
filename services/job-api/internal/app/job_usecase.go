package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"

	"github.com/cockroachdb/errors"
)

//go:generate moq -out gen_job_store_moq_test.go . JobStore
type JobStore interface {
	Jobs() JobRepository
	WithinJobTx(ctx context.Context, fn func(ctx context.Context, jobs JobRepository) error) error
}

type JobUsecase struct {
	s JobStore
}

func (u *JobUsecase) CreateJob(ctx context.Context, actor domain.Actor, name string, spec domain.JobSpec) (domain.Job, error) {
	if !actor.Role.HasCapability(domain.CapJobWrite) {
		return domain.Job{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobWrite)
	}
	var job domain.Job
	err := u.s.WithinJobTx(ctx, func(ctx context.Context, jobs JobRepository) error {
		j, err := jobs.Create(ctx, actor.ID, name, spec)
		if err != nil {
			return err
		}
		job = j
		return nil
	})
	return job, err
}

func (u *JobUsecase) UpdateJobMetadata(ctx context.Context, actor domain.Actor, job_id domain.JobId, name string, spec domain.JobSpec) (domain.Job, error) {
	if !actor.Role.HasCapability(domain.CapJobWrite) {
		return domain.Job{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobWrite)
	}
	var job domain.Job
	err := u.s.WithinJobTx(ctx, func(ctx context.Context, jobs JobRepository) error {
		err := jobs.Rename(ctx, actor.ID, job_id, name)
		if err != nil {
			return err
		}
		j, err := jobs.UpdateSpec(ctx, actor.ID, job_id, spec)
		if err != nil {
			return err
		}
		job = j
		return nil
	})
	return job, err
}

func (u *JobUsecase) SetJobEnabled(ctx context.Context, actor domain.Actor, job_id domain.JobId, enabled bool) (domain.Job, error) {
	if !actor.Role.HasCapability(domain.CapJobWrite) {
		return domain.Job{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobWrite)
	}
	var job domain.Job
	err := u.s.WithinJobTx(ctx, func(ctx context.Context, jobs JobRepository) error {
		j, err := jobs.SetEnabled(ctx, actor.ID, job_id, enabled)
		if err != nil {
			return err
		}
		job = j
		return nil
	})
	return job, err
}

func (u *JobUsecase) DeleteJob(ctx context.Context, actor domain.Actor, job_id domain.JobId) error {
	if !actor.Role.HasCapability(domain.CapJobWrite) {
		return errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobWrite)
	}
	return u.s.WithinJobTx(ctx, func(ctx context.Context, jobs JobRepository) error {
		return jobs.Delete(ctx, actor.ID, job_id)
	})
}

func (u *JobUsecase) GetJob(ctx context.Context, actor domain.Actor, job_id domain.JobId) (domain.Job, error) {
	if !actor.Role.HasCapability(domain.CapJobRead) {
		return domain.Job{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobRead)
	}
	return u.s.Jobs().FindById(ctx, actor.ID, job_id)
}

func (u *JobUsecase) ListJobs(ctx context.Context, actor domain.Actor, page_size int32, page_token string) (ListJobsResult, error) {
	if !actor.Role.HasCapability(domain.CapJobRead) {
		return ListJobsResult{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobRead)
	}
	return u.s.Jobs().ListByOwner(ctx, actor.ID, page_size, page_token)
}

func NewJobUsecase(s JobStore) *JobUsecase {
	return &JobUsecase{s: s}
}
