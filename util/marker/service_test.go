package marker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestNewService(t *testing.T) {
	s := NewService(func(fullMethod string) bool {
		return fullMethod == "ignore"
	})
	t.Run("Unary", func(t *testing.T) {
		resp, err := s.UnaryServerInterceptor()(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req interface{}) (interface{}, error) {
			s.Mark(ctx)
			return "my-resp", nil
		})
		if assert.NoError(t, err) {
			assert.Equal(t, "my-resp", resp)
		}
	})
	t.Run("Ignore", func(t *testing.T) {
		_, err := s.UnaryServerInterceptor()(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "ignore"}, func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, nil
		})
		assert.NoError(t, err)
	})
	t.Run("Stream", func(t *testing.T) {
		err := s.StreamServerInterceptor()("", &testServerStream{}, &grpc.StreamServerInfo{}, func(srv interface{}, ss grpc.ServerStream) error {
			s.Mark(ss.Context())
			return nil
		})
		assert.NoError(t, err)
	})
}
