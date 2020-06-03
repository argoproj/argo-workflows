package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	rbacmocks "github.com/argoproj/argo/server/auth/rbac/mocks"
	"github.com/argoproj/argo/server/auth/sso"
	ssomocks "github.com/argoproj/argo/server/auth/sso/mocks"
)

func TestServer_GetWFClient(t *testing.T) {
	wfClient := &fakewfclientset.Clientset{}
	kubeClient := fake.NewSimpleClientset(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "my-sa", Namespace: "my-ns"},
		Secrets:    []corev1.ObjectReference{{Name: "my-secret"}},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-secret", Namespace: "my-ns"},
		Data:       map[string][]byte{"token": {}},
	})
	t.Run("None", func(t *testing.T) {
		_, err := NewGatekeeper(Modes{}, "my-ns", wfClient, kubeClient, nil, nil, nil)
		assert.Error(t, err)
	})
	t.Run("Invalid", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Client: true}, "my-ns", wfClient, kubeClient, nil, nil, nil)
		if assert.NoError(t, err) {
			_, err := g.Context(x("invalid"))
			assert.Error(t, err)
		}
	})
	t.Run("NotAllowed", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{SSO: true}, "my-ns", wfClient, kubeClient, nil, nil, nil)
		if assert.NoError(t, err) {
			_, err := g.Context(x("Bearer "))
			assert.Error(t, err)
		}
	})
	// not possible to unit test client auth today
	t.Run("Server", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Server: true}, "my-ns", wfClient, kubeClient, nil, nil, nil)
		assert.NoError(t, err)
		ctx, err := g.Context(x(""))
		if assert.NoError(t, err) {
			assert.Equal(t, wfClient, GetWfClient(ctx))
			assert.Equal(t, kubeClient, GetKubeClient(ctx))
		}
	})
	t.Run("SSO", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&sso.Claims{}, nil)
		rbacIf := &rbacmocks.Interface{}
		rbacIf.On("ServiceAccount", mock.Anything).Return(&corev1.LocalObjectReference{Name: "my-sa"}, nil)
		g, err := NewGatekeeper(Modes{SSO: true}, "my-ns", wfClient, kubeClient, nil, ssoIf, rbacIf)
		if assert.NoError(t, err) {
			ctx, err := g.Context(x("Bearer id_token:whatever"))
			if assert.NoError(t, err) {
				assert.NotNil(t, GetWfClient(ctx))
				assert.NotNil(t, GetKubeClient(ctx))
			}
		}
	})
}

func x(authorization string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": authorization}))
}
