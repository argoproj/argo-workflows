package rbac

import (
	"context"

	"google.golang.org/grpc"
)

var NullService Service = &nullService{}

type nullService struct {
}

func (n nullService) Enforce(context.Context, string) error {
	return nil
}

func (n nullService) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		return handler(ctx, req)
	}
}

func (n nullService) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}
}
