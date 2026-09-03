package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RejectNilRequestUnaryServerInterceptor rejects nil gRPC requests.
func RejectNilRequestUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		if req == nil {
			return nil, status.Error(codes.InvalidArgument, "request is required")
		}
		return handler(ctx, req)
	}
}
