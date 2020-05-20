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

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo/server/auth/rbac"
	"github.com/argoproj/argo/server/auth/sso/mocks"
)

func TestServer_GetWFClient(t *testing.T) {
	wfClient := &fakewfclientset.Clientset{}
	kubeClient := fake.NewSimpleClientset(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-sa"},
		Secrets:    []corev1.ObjectReference{{Name: "my-secret"}},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-secret"},
		// base64("my-token") = "Im15LXRva2VuIg=="
		Data: map[string][]byte{"token": []byte("Im15LXRva2VuIg==")},
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
		ssoIf := &mocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(wfv1.User{Name: "my-name", Groups: []string{"my-group"}}, nil)
		g, err := NewGatekeeper(Modes{SSO: true}, "my-ns", nil, kubeClient, nil, ssoIf, rbac.Config{DefaultServiceAccount: &corev1.LocalObjectReference{Name: "my-sa"}})
		if assert.NoError(t, err) {
			ctx, err := g.Context(x("Bearer id_token:whatever"))
			if assert.NoError(t, err) {
				user := GetUser(ctx)
				assert.Equal(t, "my-name", user.Name)
				assert.Equal(t, []string{"my-group"}, user.Groups)
				assert.NotNil(t, GetWfClient(ctx))
				assert.NotNil(t, GetKubeClient(ctx))
			}
		}
	})
}

func x(authorization string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": authorization}))
}
