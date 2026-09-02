package auth

import (
	"context"
	"jobdeck/services/job-worker/internal/transport/interceptor"
	"testing"
	"time"

	authv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/auth/go/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type authClientStub struct {
	responses []*authv1.AuthenticateLocalClientResponse
	err       error
	requests  []*authv1.AuthenticateLocalClientRequest
}

func (c *authClientStub) AuthenticateLocalClient(_ context.Context, req *authv1.AuthenticateLocalClientRequest, _ ...grpc.CallOption) (*authv1.AuthenticateLocalClientResponse, error) {
	c.requests = append(c.requests, req)
	if c.err != nil {
		return nil, c.err
	}
	response := c.responses[len(c.requests)-1]
	return response, nil
}

func (c *authClientStub) ExchangeGoogleIdToken(context.Context, *authv1.ExchangeGoogleIdTokenRequest, ...grpc.CallOption) (*authv1.ExchangeGoogleIdTokenResponse, error) {
	return nil, nil
}

func TestCachedTokenSourceTokenCachesAndRefreshes(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC)
	client := &authClientStub{responses: []*authv1.AuthenticateLocalClientResponse{
		{Token: &authv1.Token{AccessToken: "first", TokenType: "Bearer", ExpiresInSeconds: 120}},
		{Token: &authv1.Token{AccessToken: "second", TokenType: "Bearer", ExpiresInSeconds: 120}},
		{Token: &authv1.Token{AccessToken: "third", TokenType: "Bearer", ExpiresInSeconds: 120}},
	}}
	source := &cachedTokenSource{
		now:           func() time.Time { return now },
		client:        client,
		client_id:     "worker",
		client_secret: "secret",
	}

	token, err := source.Token(context.Background())
	require.NoError(t, err)
	require.Equal(t, interceptor.AccessToken{Type: "Bearer", Value: "first"}, token)
	require.Equal(t, []*authv1.AuthenticateLocalClientRequest{{
		ClientId: "worker", ClientSecret: "secret", Audience: "jobdeck-job-api",
	}}, client.requests)

	now = now.Add(89 * time.Second)
	token, err = source.Token(context.Background())
	require.NoError(t, err)
	require.Equal(t, "first", token.Value)
	require.Len(t, client.requests, 1)

	now = now.Add(2 * time.Second)
	token, err = source.Token(context.Background())
	require.NoError(t, err)
	require.Equal(t, "second", token.Value)
	require.Len(t, client.requests, 2)

	source.Invalidate()
	token, err = source.Token(context.Background())
	require.NoError(t, err)
	require.Equal(t, "third", token.Value)
	require.Len(t, client.requests, 3)
}

func TestCachedTokenSourceTokenRejectsInvalidResponse(t *testing.T) {
	t.Parallel()

	for _, response := range []*authv1.AuthenticateLocalClientResponse{
		nil,
		{},
		{Token: &authv1.Token{TokenType: "Bearer", ExpiresInSeconds: 1}},
		{Token: &authv1.Token{AccessToken: "token", ExpiresInSeconds: 1}},
		{Token: &authv1.Token{AccessToken: "token", TokenType: "Bearer"}},
	} {
		t.Run("invalid", func(t *testing.T) {
			source := &cachedTokenSource{
				now:    time.Now,
				client: &authClientStub{responses: []*authv1.AuthenticateLocalClientResponse{response}},
			}

			_, err := source.Token(context.Background())
			require.Error(t, err)
			require.Nil(t, source.cachedToken)
		})
	}
}
