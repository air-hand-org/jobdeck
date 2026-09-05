package app

import (
	"context"

	"jobdeck/services/job-api/internal/domain"
)

// ListJobsResult contains a page of jobs.
type ListJobsResult struct {
	Jobs          []domain.Job
	NextPageToken string
}

// JobRepository persists jobs.
//
//go:generate moq -out gen_job_repository_moq_test.go . JobRepository
type JobRepository interface {
	Create(ctx context.Context, ownerID domain.ActorId, name string, spec domain.JobSpec) (domain.Job, error)
	Rename(ctx context.Context, ownerID domain.ActorId, jobID domain.JobId, name string) error
	UpdateSpec(context.Context, domain.ActorId, domain.JobId, domain.JobSpec) (domain.Job, error)
	SetEnabled(context.Context, domain.ActorId, domain.JobId, bool) (domain.Job, error)
	Delete(context.Context, domain.ActorId, domain.JobId) error

	FindByID(context.Context, domain.ActorId, domain.JobId) (domain.Job, error)
	ListByOwner(ctx context.Context, ownerID domain.ActorId, pageSize int32, pageToken string) (ListJobsResult, error)
}
