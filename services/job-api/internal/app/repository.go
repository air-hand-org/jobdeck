package app

import (
	"context"

	"jobdeck/services/job-api/internal/domain"
)

//go:generate moq -out gen_job_repository_moq_test.go . JobRepository
type JobRepository interface {
	Create(context.Context, domain.UserId, domain.JobSpec) (domain.Job, error)
	UpdateSpec(context.Context, domain.UserId, domain.JobId, domain.JobSpec) (domain.Job, error)
	SetEnabled(context.Context, domain.UserId, domain.JobId, bool) (domain.Job, error)
	Delete(context.Context, domain.UserId, domain.JobId) error

	FindById(context.Context, domain.UserId, domain.JobId) (domain.Job, error)
	ListByOwner(context.Context, domain.UserId) ([]domain.Job, error)
}
