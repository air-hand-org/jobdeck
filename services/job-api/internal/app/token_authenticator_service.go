package app

import (
	"context"
	"jobdeck/services/job-api/internal/domain"

	"github.com/cockroachdb/errors"
)

// ActorStore provides actor repositories and transactions.
//
//go:generate moq -out gen_actor_store_moq_test.go . ActorStore
type ActorStore interface {
	Actors() ActorRepository
	WithinActorTx(ctx context.Context, fn func(ctx context.Context, actors ActorRepository) error) error
}

// TokenVerifier verifies authentication tokens.
//
//go:generate moq -out gen_token_verifier_moq_test.go . TokenVerifier
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (domain.AuthenticatedIdentity, error)
}

// TokenAuthenticatorService authenticates tokens and loads actors.
type TokenAuthenticatorService struct {
	store    ActorStore
	verifier TokenVerifier
}

// Authenticate verifies token and returns the corresponding actor.
func (s *TokenAuthenticatorService) Authenticate(ctx context.Context, token string) (domain.Actor, error) {
	parsed, err := s.verifier.Verify(ctx, token)
	if err != nil {
		return domain.Actor{}, err
	}

	actor, err := s.store.Actors().GetActor(ctx, parsed.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrActorNotFound) {
			return domain.Actor{}, err
		}
		var insertedActor domain.Actor
		err := s.store.WithinActorTx(ctx, func(ctx context.Context, actors ActorRepository) error {
			actor, err := actors.CreateActor(ctx, parsed.ID, parsed.Type)
			if err != nil {
				return err
			}
			insertedActor = actor
			return nil
		})
		return insertedActor, err
	}

	if !actor.Active {
		return domain.Actor{}, domain.ErrActorInactive
	}
	return actor, nil
}

// NewTokenAuthenticatorService constructs an authenticator.
func NewTokenAuthenticatorService(store ActorStore, verifier TokenVerifier) *TokenAuthenticatorService {
	return &TokenAuthenticatorService{
		store:    store,
		verifier: verifier,
	}
}
