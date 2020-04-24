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

	At the start of the request, we create a random marker and put it into the context.

	Any sub-types should invoke the `Mark` method to indicate the expected this happened.

	At the end of the request, we check the marker and panic if it was not ticked off.
*/
type Service interface {
	Mark(ctx context.Context)
	UnaryServerInterceptor() grpc.UnaryServerInterceptor
	StreamServerInterceptor() grpc.StreamServerInterceptor
}

func NewService(ignore func(fullMethod string) bool) Service {
	return &service{struct{}{}, make(map[int]bool), ignore}
}

type service struct {
	markerKey struct{}
	markers   map[int]bool
	ignore    func(fullMethod string) bool
}

func (s *service) context(ctx context.Context) (func(), context.Context, int) {
	marker := rand.Int()
	return func() { delete(s.markers, marker) }, context.WithValue(ctx, s.markerKey, marker), marker
}

func (s *service) Mark(ctx context.Context) {
	marker, ok := ctx.Value(s.markerKey).(int)
	if ok {
		s.markers[marker] = true
	}
}

func (s *service) check(fullMethod string, marker int) {
	if s.ignore(fullMethod) {
		return
	}
	_, ok := s.markers[marker]
	logCtx := log.WithField("fullMethod", fullMethod)
	if !ok {
		logCtx.Fatal("marker not found - this should never happen")
	}
}

func (s *service) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		closer, ctx, marker := s.context(ss.Context())
		defer closer()
		wrapped := grpc_middleware.WrapServerStream(ss)
		wrapped.WrappedContext = ctx
		err := handler(srv, wrapped)
		s.check(info.FullMethod, marker)
		return err
	}
}
func (s *service) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		closer, ctx, marker := s.context(ctx)
		defer closer()
		i, err := handler(ctx, req)
		s.check(info.FullMethod, marker)
		return i, err
	}
}
