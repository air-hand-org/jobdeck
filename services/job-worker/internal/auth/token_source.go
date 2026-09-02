package auth

import (
	"context"
	"jobdeck/services/job-worker/internal/transport/interceptor"
	"time"

	authv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/auth/go/v1"
	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
)

type accessToken struct {
	Value     string
	Type      string
	ExpiresAt time.Time
}

type NowFunc func() time.Time

type cachedTokenSource struct {
	cachedToken *accessToken

	now           NowFunc
	client        authv1.AuthServiceClient
	client_id     string
	client_secret string
}

var _ interceptor.TokenSource = (*cachedTokenSource)(nil)

func (s *cachedTokenSource) Token(ctx context.Context) (interceptor.AccessToken, error) {
	if s.shouldRefreshToken() {
		token, err := s.authenticate(ctx)
		if err != nil {
			return interceptor.AccessToken{}, err
		}
		s.cachedToken = token
	}
	return interceptor.AccessToken{
		Value: s.cachedToken.Value,
		Type:  s.cachedToken.Type,
	}, nil
}

func (s *cachedTokenSource) Invalidate() {
	s.cachedToken = nil
}

func (s *cachedTokenSource) shouldRefreshToken() bool {
	if s.cachedToken == nil {
		return true
	}
	// refreshing 30 seconds before expires time
	return s.now().Add(30 * time.Second).After(s.cachedToken.ExpiresAt)
}

func validateAuthenticateLocalClientResponse(res *authv1.AuthenticateLocalClientResponse) error {
	if res == nil || res.Token == nil {
		return errors.New("failed to authenticate")
	}
	if res.Token.GetAccessToken() == "" || res.Token.GetTokenType() == "" || res.Token.GetExpiresInSeconds() <= 0 {
		return errors.New("invalid token")
	}
	return nil
}

func (s *cachedTokenSource) authenticate(ctx context.Context) (*accessToken, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	res, err := s.client.AuthenticateLocalClient(reqCtx, &authv1.AuthenticateLocalClientRequest{
		ClientId:     s.client_id,
		ClientSecret: s.client_secret,
		Audience:     "jobdeck-job-api",
	})
	if err != nil {
		return nil, err
	}
	err = validateAuthenticateLocalClientResponse(res)
	if err != nil {
		return nil, err
	}
	return &accessToken{
		Type:      res.GetToken().GetTokenType(),
		Value:     res.GetToken().GetAccessToken(),
		ExpiresAt: s.now().Add(time.Duration(res.GetToken().GetExpiresInSeconds()) * time.Second),
	}, nil
}

func NewTokenSource(conn *grpc.ClientConn, now NowFunc, client_id string, client_secret string) (*cachedTokenSource, error) {
	if client_id == "" || client_secret == "" {
		return nil, errors.New("client_id and client_secret are required")
	}
	return &cachedTokenSource{
		cachedToken:   nil,
		client:        authv1.NewAuthServiceClient(conn),
		now:           now,
		client_id:     client_id,
		client_secret: client_secret,
	}, nil
}
