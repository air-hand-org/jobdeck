package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
)

//go:generate moq -out gen_actor_repository_moq_test.go . ActorRepository
type ActorRepository interface {
	GetActor(ctx context.Context, actor_id domain.ActorId) (domain.Actor, error)
	CreateActor(ctx context.Context, actor_id domain.ActorId, actor_type domain.ActorType) (domain.Actor, error)
}
