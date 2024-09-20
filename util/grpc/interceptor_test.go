package grpc

import (
	"context"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockServerTransportStream struct {
	header metadata.MD
}

func (mockServerTransportStream) Method() string { return "" }
func (msts *mockServerTransportStream) SetHeader(md metadata.MD) error {
	msts.header = md
	return nil
}
func (mockServerTransportStream) SendHeader(md metadata.MD) error { return nil }
func (mockServerTransportStream) SetTrailer(md metadata.MD) error { return nil }

var _ grpc.ServerTransportStream = &mockServerTransportStream{}

func TestSetVersionHeaderUnaryServerInterceptor(t *testing.T) {
	version := &wfv1.Version{Version: "v3.1.0"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil }
	msts := &mockServerTransportStream{}
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), msts)
	interceptor := SetVersionHeaderUnaryServerInterceptor(*version)

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	require.NoError(t, err)
	assert.Equal(t, metadata.Pairs(ArgoVersionHeader, version.Version), msts.header)
}

type mockServerStream struct {
	header metadata.MD
}

func (msts mockServerStream) SetHeader(md metadata.MD) error {
	msts.header.Set(ArgoVersionHeader, md.Get(ArgoVersionHeader)...)
	return nil
}
func (mockServerStream) SendHeader(md metadata.MD) error { return nil }
func (mockServerStream) SetTrailer(md metadata.MD)       {}
func (mockServerStream) Context() context.Context        { return context.Background() }
func (mockServerStream) SendMsg(m any) error             { return nil }
func (mockServerStream) RecvMsg(m any) error             { return nil }

var _ grpc.ServerStream = &mockServerStream{}

func TestSetVersionHeaderStreamServerInterceptor(t *testing.T) {
	version := &wfv1.Version{Version: "v3.1.0"}
	handler := func(srv any, stream grpc.ServerStream) error { return nil }
	msts := &mockServerStream{header: metadata.New(nil)}
	interceptor := SetVersionHeaderStreamServerInterceptor(*version)

	err := interceptor(nil, msts, nil, handler)

	require.NoError(t, err)
	assert.Equal(t, metadata.Pairs(ArgoVersionHeader, version.Version), msts.header)
}
