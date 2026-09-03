package transport

import (
	"context"
	"net"
	"testing"
	"time"

	"jobdeck/services/auth/internal/app"
	"jobdeck/services/auth/internal/domain"
	"jobdeck/services/auth/internal/secret"

	authv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/auth/go/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type authRepositoryStub struct {
	client domain.LocalAuthClient
}

func (s authRepositoryStub) GetLocalAuthClient(context.Context, string) (domain.LocalAuthClient, error) {
	return s.client, nil
}

func (authRepositoryStub) UpdateLocalAuthClientSecretHash(context.Context, domain.ActorId, string) error {
	return nil
}

type authStoreStub struct {
	repository app.AuthRepository
}

func (s authStoreStub) Auths() app.AuthRepository { return s.repository }

func (s authStoreStub) WithinAuthTx(ctx context.Context, fn func(context.Context, app.AuthRepository) error) error {
	return fn(ctx, s.repository)
}

type tokenIssuerStub struct{}

func (tokenIssuerStub) Token(domain.ActorId, string, string, time.Time) (app.AccessToken, error) {
	return app.AccessToken{Token: "token", TokenType: "Bearer", ExpiresInSec: 60}, nil
}

func TestAuthServiceServerAuthenticateLocalClient(t *testing.T) {
	t.Parallel()

	factory := secret.NewPHCBuilderFactory()
	usecase := app.NewAuthUsecase(
		authStoreStub{repository: authRepositoryStub{client: domain.LocalAuthClient{
			ClientId:   "worker",
			SecretHash: factory.NewBuilder([]byte("0123456789abcdef")).PHC([]byte("secret")).String(),
			Type:       "SERVICE",
			Active:     true,
		}}},
		tokenIssuerStub{},
		factory,
	)

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	authv1.RegisterAuthServiceServer(server, NewAuthServiceServer(usecase))
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	connection, err := grpc.NewClient("passthrough:///auth", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })

	response, err := authv1.NewAuthServiceClient(connection).AuthenticateLocalClient(context.Background(), &authv1.AuthenticateLocalClientRequest{
		ClientId: "worker", ClientSecret: "secret", Audience: "jobdeck-job-api",
	})
	require.NoError(t, err)
	require.Equal(t, "token", response.GetToken().GetAccessToken())
}
