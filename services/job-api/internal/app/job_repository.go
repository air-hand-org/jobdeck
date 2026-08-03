package app

import (
	"context"

	"jobdeck/services/job-api/internal/domain"
)

type ListJobsResult struct {
	Jobs          []domain.Job
	NextPageToken string
}

//go:generate moq -out gen_job_repository_moq_test.go . JobRepository
type JobRepository interface {
	Create(ctx context.Context, owner_id domain.ActorId, name string, spec domain.JobSpec) (domain.Job, error)
	Rename(ctx context.Context, owner_id domain.ActorId, job_id domain.JobId, name string) error
	UpdateSpec(context.Context, domain.ActorId, domain.JobId, domain.JobSpec) (domain.Job, error)
	SetEnabled(context.Context, domain.ActorId, domain.JobId, bool) (domain.Job, error)
	Delete(context.Context, domain.ActorId, domain.JobId) error

	FindById(context.Context, domain.ActorId, domain.JobId) (domain.Job, error)
	ListByOwner(ctx context.Context, owner_id domain.ActorId, page_size int32, page_token string) (ListJobsResult, error)
}
