package common

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/kubernetes/fake"

	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
)

func TestServer_GetWFClient(t *testing.T) {
	wfClientset := &fakewfclientset.Clientset{}
	kubeClientset := &fake.Clientset{}
	t.Run("DisableClientAuth", func(t *testing.T) {
		s := NewServer(false, "my-ns", wfClientset, kubeClientset)
		wfClient, kubeClient, err := s.GetWFClient(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, wfClient, wfClientset)
		assert.Equal(t, kubeClient, kubeClientset)
	})
	t.Run("ClientAuth", func(t *testing.T) {
		s := NewServer(true, "my-ns", wfClientset, kubeClientset)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("grpcgateway-authorization", base64.StdEncoding.EncodeToString([]byte("{}"))))
		wfClient, kubeClient, err := s.GetWFClient(ctx)
		assert.NoError(t, err)
		assert.NotEqual(t, wfClient, wfClientset)
		assert.NotEqual(t, kubeClient, kubeClientset)
	})
	t.Run("Localhost", func(t *testing.T) {
		s := NewServer(true, "my-ns", wfClientset, kubeClientset)
		for _, text := range []string{
			`{"caFile": "anything"}`,
			`{"certFile": "anything"}`,
			`{"keyFile": "anything"}`,
			`{"bearerTokenFile": "anything"}`,
			`{"host": "localhost:443"}`,
			`{"host": "127.0.0.1:443"}`,
		} {
			t.Run(text, func(t *testing.T) {
				ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("grpcgateway-authorization", base64.StdEncoding.EncodeToString([]byte(text))))
				_, _, err := s.GetWFClient(ctx)
				assert.EqualError(t, err, "illegal bearer token")
			})
		}
	})
}