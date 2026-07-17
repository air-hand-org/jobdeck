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

func (r *actorRepository) GetActor(ctx context.Context, actor_id domain.ActorId) (domain.Actor, error) {
	actor, err := r.q.GetActor(ctx, actor_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Actor{}, domain.ErrActorNotFound
		}
		return domain.Actor{}, err
	}

	return actorFromDB(actor)
}

func (r *actorRepository) CreateActor(ctx context.Context, actor_id domain.ActorId, actor_type domain.ActorType) (domain.Actor, error) {
	actor_id, err := r.q.CreateActor(ctx, jobdb.CreateActorParams{
		ActorID: actor_id,
		Type:    string(actor_type),
	})
	if err != nil {
		return domain.Actor{}, err
	}

	return r.GetActor(ctx, actor_id)
}

func newActorRepository(q actorQuerier) *actorRepository {
	return &actorRepository{q: q}
}
