package grpc

import (
	"context"
	"errors"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockServerTransportStream struct {
	header  metadata.MD
	isError bool
}

func (mockServerTransportStream) Method() string { return "" }
func (msts *mockServerTransportStream) SetHeader(md metadata.MD) error {
	if msts.isError {
		return errors.New("simulate error setting header")
	}
	msts.header = md
	return nil
}
func (mockServerTransportStream) SendHeader(md metadata.MD) error { return nil }
func (mockServerTransportStream) SetTrailer(md metadata.MD) error { return nil }

var _ grpc.ServerTransportStream = &mockServerTransportStream{}

func TestSetVersionHeaderUnaryServerInterceptor(t *testing.T) {
	version := &wfv1.Version{Version: "v3.1.0"}
	mockReturn := "successful return"

	t.Run("success", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) { return mockReturn, nil }
		msts := &mockServerTransportStream{}
		baseCtx := logging.TestContext(t.Context())
		ctx := grpc.NewContextWithServerTransportStream(baseCtx, msts)
		interceptor := SetVersionHeaderUnaryServerInterceptor(*version)

		m, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

		require.NoError(t, err)
		assert.Equal(t, mockReturn, m)
		assert.Equal(t, metadata.Pairs(ArgoVersionHeader, version.Version), msts.header)
	})

	t.Run("upstream error handling", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, errors.New("error") }
		msts := &mockServerTransportStream{}
		baseCtx := logging.TestContext(t.Context())
		ctx := grpc.NewContextWithServerTransportStream(baseCtx, msts)
		interceptor := SetVersionHeaderUnaryServerInterceptor(*version)

		_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

		require.Error(t, err)
		assert.Empty(t, msts.header)
	})

	t.Run("SetHeader error handling", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return mockReturn, nil
		}
		msts := &mockServerTransportStream{isError: true}
		baseCtx := logging.TestContext(t.Context())
		ctx := grpc.NewContextWithServerTransportStream(baseCtx, msts)
		interceptor := SetVersionHeaderUnaryServerInterceptor(*version)

		m, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

		require.NoError(t, err)
		require.Equal(t, mockReturn, m)
		assert.Empty(t, msts.header)
	})
}

type mockServerStream struct {
	header  metadata.MD
	isError bool
}

func (msts mockServerStream) SetHeader(md metadata.MD) error {
	if msts.isError {
		return errors.New("simulate error setting header")
	}
	msts.header.Set(ArgoVersionHeader, md.Get(ArgoVersionHeader)...)
	return nil
}
func (mockServerStream) SendHeader(md metadata.MD) error { return nil }
func (mockServerStream) SetTrailer(md metadata.MD)       {}
func (mockServerStream) Context() context.Context {
	// nolint:contextcheck
	return logging.TestContext(context.Background())
}
func (mockServerStream) SendMsg(m any) error { return nil }
func (mockServerStream) RecvMsg(m any) error { return nil }

var _ grpc.ServerStream = &mockServerStream{}

func TestSetVersionHeaderStreamServerInterceptor(t *testing.T) {
	version := &wfv1.Version{Version: "v3.1.0"}

	t.Run("success", func(t *testing.T) {
		handler := func(srv any, stream grpc.ServerStream) error { return nil }
		msts := &mockServerStream{header: metadata.New(nil)}
		interceptor := SetVersionHeaderStreamServerInterceptor(*version)

		err := interceptor(nil, msts, nil, handler)

		require.NoError(t, err)
		assert.Equal(t, metadata.Pairs(ArgoVersionHeader, version.Version), msts.header)
	})

	t.Run("upstream error handling", func(t *testing.T) {
		handler := func(srv any, stream grpc.ServerStream) error {
			return errors.New("test error")
		}
		msts := &mockServerStream{header: metadata.New(nil)}
		interceptor := SetVersionHeaderStreamServerInterceptor(*version)

		err := interceptor(nil, msts, nil, handler)

		require.Error(t, err, "test error")
		assert.Empty(t, msts.header)
	})

	t.Run("SetHeader error handling", func(t *testing.T) {
		handler := func(srv any, stream grpc.ServerStream) error { return nil }
		msts := &mockServerStream{isError: true}
		interceptor := SetVersionHeaderStreamServerInterceptor(*version)

		err := interceptor(nil, msts, nil, handler)

		require.NoError(t, err)
		assert.Empty(t, msts.header)
	})
}
