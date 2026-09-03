// Package interceptor contains gRPC server interceptors.
package interceptor

import (
	"context"
	"os"
	"strings"

	internal_auth "jobdeck/services/job-api/internal/auth"
	"jobdeck/services/job-api/internal/domain"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var developmentActor = func() domain.Actor {
	actor, err := domain.NewActor(
		uuid.Max,
		true,
		domain.ActorTypeUser,
		"",
	)
	if err != nil {
		panic(err)
	}
	return actor
}()

// IdentityFromTokenFunc resolves an actor from an authentication token.
type IdentityFromTokenFunc func(ctx context.Context, token string) (domain.Actor, error)

func newAuthFunc(idFromTokenFn IdentityFromTokenFunc) func(context.Context) (context.Context, error) {
	isAuthModeOptional := strings.EqualFold(os.Getenv("API_AUTH_MODE"), "OPTIONAL")
	return func(ctx context.Context) (context.Context, error) {
		if isAuthModeOptional {
			md, _ := metadata.FromIncomingContext(ctx)
			hasAuthorization := len(md.Get("authorization")) > 0
			if !hasAuthorization {
				return internal_auth.SetActorContext(ctx, developmentActor), nil
			}
		}

		token, err := auth.AuthFromMD(ctx, "bearer")
		if err != nil {
			return nil, err
		}

		identity, err := idFromTokenFn(ctx, token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
		}

		return internal_auth.SetActorContext(ctx, identity), nil
	}
}

// AuthenticationInterceptor authenticates requests selected by matcher.
func AuthenticationInterceptor(matcher selector.Matcher, fn IdentityFromTokenFunc) grpc.UnaryServerInterceptor {
	return selector.UnaryServerInterceptor(
		auth.UnaryServerInterceptor(newAuthFunc(fn)),
		matcher,
	)
}
