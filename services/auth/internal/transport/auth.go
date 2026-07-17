package transport

import (
	"context"
	authv1 "jobdeck/gen/go/jobdeck/auth/v1"
	"jobdeck/services/auth/internal/app"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authSeviceServer struct {
	usecase app.AuthUsecase

	// TODO: switch to UnsafeAuthServiceServer
	authv1.UnimplementedAuthServiceServer
}

var _ authv1.AuthServiceServer = (*authSeviceServer)(nil)

func (s *authSeviceServer) AuthenticateLocalClient(ctx context.Context, req *authv1.AuthenticateLocalClientRequest) (*authv1.AuthenticateLocalClientResponse, error) {
	token, err := s.usecase.Authenticate(ctx, req.GetClientId(), req.GetClientSecret(), req.GetAudience())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "failed to authenticate")
	}
	return &authv1.AuthenticateLocalClientResponse{
		Token: &authv1.Token{
			AccessToken:      token.Token,
			TokenType:        token.TokenType,
			ExpiresInSeconds: token.ExpiresInSec,
		},
	}, nil
}

func NewAuthServiceServer(usecase *app.AuthUsecase) *authSeviceServer {
	return &authSeviceServer{
		usecase: *usecase,
	}
}
