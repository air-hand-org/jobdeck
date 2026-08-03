package app_test

import (
	"context"
	"jobdeck/services/auth/internal/app"
	"jobdeck/services/auth/internal/domain"
	"jobdeck/services/auth/internal/secret"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type authRepositoryStub struct {
	client  domain.LocalAuthClient
	err     error
	updates int
}

func (s *authRepositoryStub) GetLocalAuthClient(context.Context, string) (domain.LocalAuthClient, error) {
	return s.client, s.err
}

func (s *authRepositoryStub) UpdateLocalAuthClientSecretHash(context.Context, domain.ActorId, string) error {
	s.updates++
	return nil
}

type authStoreStub struct {
	repository app.AuthRepository
}

func (s authStoreStub) Auths() app.AuthRepository {
	return s.repository
}

func (s authStoreStub) WithinAuthTx(ctx context.Context, fn func(context.Context, app.AuthRepository) error) error {
	return fn(ctx, s.repository)
}

type tokenIssuerStub struct {
	calls int
}

func (s *tokenIssuerStub) Token(domain.ActorId, string, string, time.Time) (app.AccessToken, error) {
	s.calls++
	return app.AccessToken{Token: "token", TokenType: "Bearer", ExpiresInSec: 60}, nil
}

func localAuthClient(active bool, password string) domain.LocalAuthClient {
	factory := secret.NewPHCBuilderFactory()
	return domain.LocalAuthClient{
		ActorId:    uuid.New(),
		ClientId:   "client",
		SecretHash: factory.NewBuilder([]byte("0123456789abcdef")).PHC([]byte(password)).String(),
		Type:       "SERVICE",
		Active:     active,
	}
}

func TestAuthUsecaseAuthenticateRejectsInactiveClient(t *testing.T) {
	t.Parallel()

	repository := &authRepositoryStub{client: localAuthClient(false, "secret")}
	issuer := &tokenIssuerStub{}
	usecase := app.NewAuthUsecase(authStoreStub{repository: repository}, issuer, secret.NewPHCBuilderFactory())

	_, err := usecase.Authenticate(context.Background(), "client", "secret", "jobdeck-job-api")

	require.Error(t, err)
	require.Zero(t, issuer.calls)
	require.Zero(t, repository.updates)
}

func TestAuthUsecaseAuthenticateRejectsInvalidSecret(t *testing.T) {
	t.Parallel()

	repository := &authRepositoryStub{client: localAuthClient(true, "secret")}
	issuer := &tokenIssuerStub{}
	usecase := app.NewAuthUsecase(authStoreStub{repository: repository}, issuer, secret.NewPHCBuilderFactory())

	_, err := usecase.Authenticate(context.Background(), "client", "incorrect", "jobdeck-job-api")

	require.Error(t, err)
	require.Zero(t, issuer.calls)
	require.Zero(t, repository.updates)
}

func TestAuthUsecaseAuthenticateReturnsIssuerToken(t *testing.T) {
	t.Parallel()

	repository := &authRepositoryStub{client: localAuthClient(true, "secret")}
	issuer := &tokenIssuerStub{}
	usecase := app.NewAuthUsecase(authStoreStub{repository: repository}, issuer, secret.NewPHCBuilderFactory())

	token, err := usecase.Authenticate(context.Background(), "client", "secret", "jobdeck-job-api")

	require.NoError(t, err)
	require.Equal(t, "token", token.Token)
	require.Equal(t, 1, issuer.calls)
	require.Zero(t, repository.updates)
}
