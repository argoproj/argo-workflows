package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo/server/auth/oauth2/mocks"
)

func TestServer_GetWFClient(t *testing.T) {
	wfClient := &fakewfclientset.Clientset{}
	kubeClient := &fake.Clientset{}
	t.Run("NoAuth", func(t *testing.T) {
		_, err := NewGatekeeper(Modes{}, wfClient, kubeClient, nil, nil)
		assert.Error(t, err)
	})
	t.Run("SSO", func(t *testing.T) {
		oauth2Service := &mocks.Service{}
		oauth2Service.On("IsSSO", mock.Anything).Return(true)
		oauth2Service.On("Authorize", mock.Anything, mock.Anything).Return(wfv1.User{Name: "my-name"}, nil)
		s, err := NewGatekeeper(Modes{SSO: true}, wfClient, kubeClient, nil, oauth2Service)
		if assert.NoError(t, err) {
			ctx, err := s.Context(context.Background())
			if assert.NoError(t, err) {
				user := GetUser(ctx)
				assert.Equal(t, "my-name", user.Name)
			}
		}
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

func authAndHandle(s Gatekeeper, ctx context.Context) (*context.Context, error) {
	var usedCtx *context.Context
	_, err := s.UnaryServerInterceptor()(ctx, nil, nil, func(ctx context.Context, req interface{}) (i interface{}, err error) {
		usedCtx = &ctx
		return nil, nil
	})
	return usedCtx, err
}
