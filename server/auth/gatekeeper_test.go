package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"

	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
)

func TestServer_GetWFClient(t *testing.T) {
	wfClient := &fakewfclientset.Clientset{}
	kubeClient := &fake.Clientset{}

	t.Run("NoAuth", func(t *testing.T) {
		_, err := NewGatekeeper(Modes{}, wfClient, kubeClient, nil, nil)
		assert.Error(t, err)
	})
	t.Run("ServerAuth", func(t *testing.T) {
		s, err := NewGatekeeper(Modes{Server: true}, wfClient, kubeClient, nil, nil)
		assert.NoError(t, err)
		ctx, err := authAndHandle(s, context.TODO())
		if assert.NoError(t, err) {
			assert.Equal(t, wfClient, GetWfClient(*ctx))
			assert.Equal(t, kubeClient, GetKubeClient(*ctx))
		}
	})
}

func authAndHandle(s *Gatekeeper, ctx context.Context) (*context.Context, error) {
	var usedCtx *context.Context
	_, err := s.UnaryServerInterceptor()(ctx, nil, nil, func(ctx context.Context, req interface{}) (i interface{}, err error) {
		usedCtx = &ctx
		return nil, nil
	})
	return usedCtx, err
}
