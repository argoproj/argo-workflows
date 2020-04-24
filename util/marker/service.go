package marker

import (
	"context"
	"math/rand"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

/*
	A marker service is an abstract service.

	1. At the start of the request, `ContextWithMarker` creates a random marker and puts it into the context.
	2. Any sub-types then invokes `Mark`  to indicate the expected this happened.
	3. At the end of the request `Check` checks the marker and panics if it was not marked.
*/
type Service interface {
	ContextWithMarker(ctx context.Context) (func(), context.Context, int)
	Mark(ctx context.Context)
	Check(fullMethod string, marker int)
	UnaryServerInterceptor() grpc.UnaryServerInterceptor
	StreamServerInterceptor() grpc.StreamServerInterceptor
}

func NewService(ignore func(fullMethod string) bool) Service {
	return &service{struct{}{}, make(map[int]bool), ignore}
}

type service struct {
	markerKey struct{}
	markers   map[int]bool
	ignore    func(operation string) bool
}

func (s *service) ContextWithMarker(ctx context.Context) (func(), context.Context, int) {
	marker := rand.Int()
	return func() { delete(s.markers, marker) }, context.WithValue(ctx, s.markerKey, marker), marker
}

func (s *service) Mark(ctx context.Context) {
	marker, ok := ctx.Value(s.markerKey).(int)
	if ok {
		s.markers[marker] = true
	}
}

func (s *service) Check(operation string, marker int) {
	if s.ignore(operation) {
		return
	}
	_, ok := s.markers[marker]
	if !ok {
		log.WithField("operation", operation).Fatal("marker not found - this should never happen")
	}
}

func (s *service) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		closer, ctx, marker := s.ContextWithMarker(ss.Context())
		defer closer()
		wrapped := grpc_middleware.WrapServerStream(ss)
		wrapped.WrappedContext = ctx
		err := handler(srv, wrapped)
		s.Check(info.FullMethod, marker)
		return err
	}
}
func (s *service) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		closer, ctx, marker := s.ContextWithMarker(ctx)
		defer closer()
		i, err := handler(ctx, req)
		s.Check(info.FullMethod, marker)
		return i, err
	}
}
