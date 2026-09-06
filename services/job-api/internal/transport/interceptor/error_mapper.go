package interceptor

import (
	"context"
	"jobdeck/services/job-api/internal/domain"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorMappingUnaryServerInterceptor maps domain errors to gRPC status errors.
func ErrorMappingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return nil, toGrpcError(err)
		}
		return resp, nil
	}
}

func toGrpcError(err error) error {
	// no transform if err already grpc.status.Error
	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "timeout")
	case errors.Is(err, domain.ErrPermissionDenied):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, domain.ErrUnauthenticated):
		return status.Error(codes.Unauthenticated, "authenticate required")
	case errors.Is(err, domain.ErrExecutionHasAlreadyConsumed):
		return status.Error(codes.FailedPrecondition, "conflicted")
	case errors.Is(err, domain.ErrExecutionNotFound), errors.Is(err, domain.ErrJobNotFound):
		return status.Error(codes.NotFound, "not found")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
