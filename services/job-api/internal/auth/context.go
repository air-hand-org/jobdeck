package auth

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
)

type identityKey struct{}

func GetActorContext(ctx context.Context) (domain.Actor, bool) {
	actor, ok := ctx.Value(identityKey{}).(domain.Actor)
	return actor, ok
}

func SetActorContext(ctx context.Context, actor domain.Actor) context.Context {
	return context.WithValue(ctx, identityKey{}, actor)
}
