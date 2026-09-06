// Package auth provides authentication context and token verification.
package auth

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
)

type identityKey struct{}

// GetActorContext returns the actor stored in ctx, if any.
func GetActorContext(ctx context.Context) (domain.Actor, bool) {
	actor, ok := ctx.Value(identityKey{}).(domain.Actor)
	return actor, ok
}

// SetActorContext returns a context containing actor.
func SetActorContext(ctx context.Context, actor domain.Actor) context.Context {
	return context.WithValue(ctx, identityKey{}, actor)
}
