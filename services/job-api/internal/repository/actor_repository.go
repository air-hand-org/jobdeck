// Package repository implements persistence for the job API.
package repository

import (
	"context"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	jobdb "jobdeck/services/job-api/internal/repository/gen"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type actorQuerier interface {
	CreateActor(ctx context.Context, arg jobdb.CreateActorParams) (uuid.UUID, error)
	GetActor(ctx context.Context, actorID uuid.UUID) (jobdb.JobActor, error)
}

var _ actorQuerier = (*jobdb.Queries)(nil)

type actorRepository struct {
	q actorQuerier
}

var _ app.ActorRepository = (*actorRepository)(nil)

func (r *actorRepository) GetActor(ctx context.Context, actorID domain.ActorId) (domain.Actor, error) {
	actor, err := r.q.GetActor(ctx, actorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Actor{}, domain.ErrActorNotFound
		}
		return domain.Actor{}, err
	}

	return actorFromDB(actor)
}

func (r *actorRepository) CreateActor(ctx context.Context, actorID domain.ActorId, actorType domain.ActorType) (domain.Actor, error) {
	createdActorID, err := r.q.CreateActor(ctx, jobdb.CreateActorParams{
		ActorID: actorID,
		Type:    string(actorType),
	})
	if err != nil {
		return domain.Actor{}, err
	}

	return r.GetActor(ctx, createdActorID)
}

func newActorRepository(q actorQuerier) *actorRepository {
	return &actorRepository{q: q}
}
