package interceptor

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AccessToken struct {
	Value string
	Type  string
}

func (t AccessToken) AuthorizationValue() string {
	return fmt.Sprintf("%s %s", t.Type, t.Value)
}

type TokenSource interface {
	Token(ctx context.Context) (AccessToken, error)
	Invalidate()
}

func AuthUnaryInterceptor(source TokenSource) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		call := func(ctx context.Context) error {
			token, err := source.Token(ctx)
			if err != nil {
				return err
			}
			ctx = metadata.AppendToOutgoingContext(
				ctx,
				"authorization",
				token.AuthorizationValue(),
			)
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		err := call(ctx)
		if status.Code(err) != codes.Unauthenticated {
			return err
		}

		// retry just once if unauthenticated error occured
		source.Invalidate()
		return call(ctx)
	}
}
