package grpc

import (
	"context"
	"runtime/debug"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	limiter "github.com/sethvargo/go-limiter"
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

// RatelimitUnaryServerInterceptor returns a new unary server interceptor that performs request rate limiting.
func RatelimitUnaryServerInterceptor(ratelimiter limiter.Store) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ip := getClientIP(ctx)
		_, _, _, ok, err := ratelimiter.Take(ctx, ip)
		if err != nil {
			log.Warnf("Internal Server Error: %s", err)
			return nil, status.Errorf(codes.Internal, "%s: grpc_ratelimit middleware internal error", info.FullMethod)
		}
		if !ok {
			return nil, status.Errorf(codes.ResourceExhausted, "%s is rejected by grpc_ratelimit middleware, please retry later.", info.FullMethod)
		}
		return handler(ctx, req)
	}
}

// RatelimitStreamServerInterceptor returns a new stream server interceptor that performs rate limiting on the request.
func RatelimitStreamServerInterceptor(ratelimiter limiter.Store) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		ip := getClientIP(ctx)
		_, _, _, ok, err := ratelimiter.Take(ctx, ip)
		if err != nil {
			log.Warnf("Internal Server Error: %s", err)
			return status.Errorf(codes.Internal, "%s: grpc_ratelimit middleware internal error", info.FullMethod)
		}
		if !ok {
			return status.Errorf(codes.ResourceExhausted, "%s is rejected by grpc_ratelimit middleware, please retry later.", info.FullMethod)
		}
		return handler(srv, stream)
	}
}

// GetClientIP inspects the context to retrieve the ip address of the client
func getClientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		log.Warnf("couldn't parse client IP address")
		return ""
	}
	address := p.Addr.String()
	ip := strings.Split(address, ":")[0]
	return ip
}
