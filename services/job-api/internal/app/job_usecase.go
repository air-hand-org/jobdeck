package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"

	"github.com/cockroachdb/errors"
)

// JobStore provides job repositories and transactions.
//
//go:generate moq -out gen_job_store_moq_test.go . JobStore
type JobStore interface {
	Jobs() JobRepository
	WithinJobTx(ctx context.Context, fn func(ctx context.Context, jobs JobRepository) error) error
}

// JobUsecase implements job operations.
type JobUsecase struct {
	s JobStore
}

// CreateJob creates a job.
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

// UpdateJobMetadata updates a job name and specification.
func (u *JobUsecase) UpdateJobMetadata(ctx context.Context, actor domain.Actor, jobID domain.JobId, name string, spec domain.JobSpec) (domain.Job, error) {
	if !actor.Role.HasCapability(domain.CapJobWrite) {
		return domain.Job{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobWrite)
	}
	var job domain.Job
	err := u.s.WithinJobTx(ctx, func(ctx context.Context, jobs JobRepository) error {
		err := jobs.Rename(ctx, actor.ID, jobID, name)
		if err != nil {
			return err
		}
		j, err := jobs.UpdateSpec(ctx, actor.ID, jobID, spec)
		if err != nil {
			return err
		}
		job = j
		return nil
	})
	return job, err
}

// SetJobEnabled enables or disables a job.
func (u *JobUsecase) SetJobEnabled(ctx context.Context, actor domain.Actor, jobID domain.JobId, enabled bool) (domain.Job, error) {
	if !actor.Role.HasCapability(domain.CapJobWrite) {
		return domain.Job{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobWrite)
	}
	var job domain.Job
	err := u.s.WithinJobTx(ctx, func(ctx context.Context, jobs JobRepository) error {
		j, err := jobs.SetEnabled(ctx, actor.ID, jobID, enabled)
		if err != nil {
			return err
		}
		job = j
		return nil
	})
	return job, err
}

// DeleteJob deletes a job.
func (u *JobUsecase) DeleteJob(ctx context.Context, actor domain.Actor, jobID domain.JobId) error {
	if !actor.Role.HasCapability(domain.CapJobWrite) {
		return errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobWrite)
	}
	return u.s.WithinJobTx(ctx, func(ctx context.Context, jobs JobRepository) error {
		return jobs.Delete(ctx, actor.ID, jobID)
	})
}

// GetJob retrieves a job.
func (u *JobUsecase) GetJob(ctx context.Context, actor domain.Actor, jobID domain.JobId) (domain.Job, error) {
	if !actor.Role.HasCapability(domain.CapJobRead) {
		return domain.Job{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobRead)
	}
	return u.s.Jobs().FindByID(ctx, actor.ID, jobID)
}

// ListJobs lists jobs for an actor.
func (u *JobUsecase) ListJobs(ctx context.Context, actor domain.Actor, pageSize int32, pageToken string) (ListJobsResult, error) {
	if !actor.Role.HasCapability(domain.CapJobRead) {
		return ListJobsResult{}, errors.Wrapf(domain.ErrPermissionDenied, "actor %s has no cap %s", actor.ID, domain.CapJobRead)
	}
	return u.s.Jobs().ListByOwner(ctx, actor.ID, pageSize, pageToken)
}

// NewJobUsecase constructs a job use case.
func NewJobUsecase(s JobStore) *JobUsecase {
	return &JobUsecase{s: s}
}
