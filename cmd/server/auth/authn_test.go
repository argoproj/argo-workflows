package auth

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/kubernetes/fake"

	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
)

func TestServer_GetWFClient(t *testing.T) {
	wfClient := &fakewfclientset.Clientset{}
	kubeClient := &fake.Clientset{}
	t.Run("DisableClientAuth", func(t *testing.T) {
		s := NewAuthN(false, wfClient, kubeClient)
		ctx, err := authAndHandle(s, context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, wfClient, GetWfClient(*ctx))
		assert.Equal(t, kubeClient, GetKubeClient(*ctx))
	})
	t.Run("ClientAuth", func(t *testing.T) {
		s := NewAuthN(true, wfClient, kubeClient)
		ctx, err := authAndHandle(s, metadata.NewIncomingContext(context.Background(), metadata.Pairs("grpcgateway-authorization", base64.StdEncoding.EncodeToString([]byte("{}")))))
		assert.NoError(t, err)
		assert.NotEqual(t, wfClient, GetWfClient(*ctx))
		assert.NotEqual(t, kubeClient, GetKubeClient(*ctx))
	})
	t.Run("Localhost", func(t *testing.T) {
		 s := NewAuthN(true, wfClient, kubeClient)
		for _, text := range []string{
			`{"caFile": "anything"}`,
			`{"certFile": "anything"}`,
			`{"keyFile": "anything"}`,
			`{"bearerTokenFile": "anything"}`,
			`{"host": "localhost:443"}`,
			`{"host": "127.0.0.1:443"}`,
		} {
			t.Run(text, func(t *testing.T) {
				_, err := authAndHandle(s, metadata.NewIncomingContext(context.Background(), metadata.Pairs("grpcgateway-authorization", base64.StdEncoding.EncodeToString([]byte(text)))))
				assert.Error(t, err)
				assert.Equal(t, codes.Unauthenticated, status.Code(err))
			})
		}
	})
}

func authAndHandle(s AuthN, ctx context.Context) (*context.Context, error) {
	var usedCtx *context.Context
	_, err := s.UnaryServerInterceptor()(ctx, nil, nil, func(ctx context.Context, req interface{}) (i interface{}, err error) {
		usedCtx = &ctx
		return nil, nil
	})
	return usedCtx, err
}
