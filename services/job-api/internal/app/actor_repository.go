// Package app contains application use cases and ports.
package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
)

// ActorRepository persists actors.
//
//go:generate moq -out gen_actor_repository_moq_test.go . ActorRepository
type ActorRepository interface {
	GetActor(ctx context.Context, actorID domain.ActorId) (domain.Actor, error)
	CreateActor(ctx context.Context, actorID domain.ActorId, actorType domain.ActorType) (domain.Actor, error)
}
