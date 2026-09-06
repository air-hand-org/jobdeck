package interceptor

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
)

// LoggingErrorUnaryServerInterceptor logs errors returned by handlers.
func LoggingErrorUnaryServerInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			logger.ErrorContext(
				ctx,
				"gRPC request failed",
				"method", info.FullMethod,
				"error", err,
			)
			return nil, err
		}
		return resp, nil
	}
}
