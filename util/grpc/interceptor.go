package grpc

import (
	"runtime/debug"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PanicLoggerUnaryServerInterceptor returns a new unary server interceptor for recovering from panics and returning error
func PanicLoggerUnaryServerInterceptor(log *log.Entry) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Recovered from panic: %+v\n%s", r, debug.Stack())
				err = status.Errorf(codes.Internal, "%s", r)
			}
		}()
		return handler(ctx, req)
	}
}

// PanicLoggerStreamServerInterceptor returns a new streaming server interceptor for recovering from panics and returning error
func PanicLoggerStreamServerInterceptor(log *log.Entry) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Recovered from panic: %+v\n%s", r, debug.Stack())
				err = status.Errorf(codes.Internal, "%s", r)
			}
		}()
		return handler(srv, stream)
	}
}

var (
	ErrorTranslationUnaryServerInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		resp, err = handler(ctx, req)
		return resp, TranslateError(err)
	}
	ErrorTranslationStreamServerInterceptor = func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return TranslateError(handler(srv, ss))
	}
)
