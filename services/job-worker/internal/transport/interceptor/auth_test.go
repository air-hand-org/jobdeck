package interceptor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type tokenSourceStub struct {
	tokens          []AccessToken
	tokenCalls      int
	invalidateCalls int
	err             error
}

func (s *tokenSourceStub) Token(context.Context) (AccessToken, error) {
	if s.err != nil {
		return AccessToken{}, s.err
	}
	token := s.tokens[s.tokenCalls]
	s.tokenCalls++
	return token, nil
}

func (s *tokenSourceStub) Invalidate() {
	s.invalidateCalls++
}

func TestAuthUnaryInterceptorAddsAuthorizationHeader(t *testing.T) {
	t.Parallel()

	source := &tokenSourceStub{tokens: []AccessToken{{Type: "Bearer", Value: "token"}}}
	interceptor := AuthUnaryInterceptor(source)

	err := interceptor(context.Background(), "/jobdeck.JobService/ListJobs", nil, nil, nil, func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		metadata, found := metadata.FromOutgoingContext(ctx)
		require.True(t, found)
		values := metadata.Get("authorization")
		require.Equal(t, []string{"Bearer token"}, values)
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 1, source.tokenCalls)
	require.Zero(t, source.invalidateCalls)
}

func TestAuthUnaryInterceptorRetriesUnauthenticatedOnce(t *testing.T) {
	t.Parallel()

	source := &tokenSourceStub{tokens: []AccessToken{
		{Type: "Bearer", Value: "expired"},
		{Type: "Bearer", Value: "refreshed"},
	}}
	interceptor := AuthUnaryInterceptor(source)
	var authorizations []string
	calls := 0

	err := interceptor(context.Background(), "/jobdeck.JobService/ListJobs", nil, nil, nil, func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		metadata, found := metadata.FromOutgoingContext(ctx)
		require.True(t, found)
		authorizations = append(authorizations, metadata.Get("authorization")[0])
		calls++
		if calls == 1 {
			return status.Error(codes.Unauthenticated, "expired")
		}
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, []string{"Bearer expired", "Bearer refreshed"}, authorizations)
	require.Equal(t, 2, source.tokenCalls)
	require.Equal(t, 1, source.invalidateCalls)
}

func TestAuthUnaryInterceptorDoesNotRetryOtherErrors(t *testing.T) {
	t.Parallel()

	source := &tokenSourceStub{tokens: []AccessToken{{Type: "Bearer", Value: "token"}}}
	interceptor := AuthUnaryInterceptor(source)
	expected := errors.New("connection failed")

	err := interceptor(context.Background(), "/jobdeck.JobService/ListJobs", nil, nil, nil, func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		return expected
	})

	require.ErrorIs(t, err, expected)
	require.Equal(t, 1, source.tokenCalls)
	require.Zero(t, source.invalidateCalls)
}
