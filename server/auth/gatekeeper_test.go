package auth

import (
	"context"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
	"gopkg.in/square/go-jose.v2/jwt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	ssomocks "github.com/argoproj/argo/server/auth/sso/mocks"
	"github.com/argoproj/argo/server/auth/types"
	"github.com/argoproj/argo/workflow/common"
)

func TestServer_GetWFClient(t *testing.T) {
	// prevent using local KUBECONFIG - which will fail on CI
	_ = os.Setenv("KUBECONFIG", "/dev/null")
	defer func() { _ = os.Unsetenv("KUBECONFIG") }()
	wfClient := fakewfclientset.NewSimpleClientset()
	kubeClient := kubefake.NewSimpleClientset(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-other-sa", Namespace: "my-ns",
				Annotations: map[string]string{
					common.AnnotationKeyRBACRule:           "'other-group' in groups",
					common.AnnotationKeyRBACRulePrecedence: "0",
				},
			},
			Secrets: []corev1.ObjectReference{{Name: "my-secret"}},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-sa", Namespace: "my-ns",
				Annotations: map[string]string{
					common.AnnotationKeyRBACRule:           "'my-group' in groups",
					common.AnnotationKeyRBACRulePrecedence: "1",
				},
			},
			Secrets: []corev1.ObjectReference{{Name: "my-secret"}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "my-secret", Namespace: "my-ns"},
			Data: map[string][]byte{
				"token": {},
			},
		},
	)
	var clientForAuthorization ClientForAuthorization = func(authorization string) (*rest.Config, versioned.Interface, kubernetes.Interface, error) {
		return &rest.Config{}, &fakewfclientset.Clientset{}, &kubefake.Clientset{}, nil
	}
	t.Run("None", func(t *testing.T) {
		_, err := NewGatekeeper(Modes{}, wfClient, kubeClient, nil, nil, clientForAuthorization, "")
		assert.Error(t, err)
	})
	t.Run("Invalid", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Client: true}, wfClient, kubeClient, nil, nil, clientForAuthorization, "")
		if assert.NoError(t, err) {
			_, err := g.Context(x("invalid"))
			assert.Error(t, err)
		}
	})
	t.Run("NotAllowed", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{SSO: true}, wfClient, kubeClient, nil, nil, clientForAuthorization, "")
		if assert.NoError(t, err) {
			_, err := g.Context(x("Bearer "))
			assert.Error(t, err)
		}
	})
	t.Run("Client", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Client: true}, wfClient, kubeClient, &rest.Config{Username: "my-username"}, nil, clientForAuthorization, "")
		assert.NoError(t, err)
		ctx, err := g.Context(x("Bearer "))
		if assert.NoError(t, err) {
			assert.NotEqual(t, wfClient, GetWfClient(ctx))
			assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
			assert.Nil(t, GetClaims(ctx))
		}
	})
	t.Run("Server", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Server: true}, wfClient, kubeClient, &rest.Config{Username: "my-username"}, nil, clientForAuthorization, "")
		assert.NoError(t, err)
		ctx, err := g.Context(x(""))
		if assert.NoError(t, err) {
			assert.Equal(t, wfClient, GetWfClient(ctx))
			assert.Equal(t, kubeClient, GetKubeClient(ctx))
			assert.NotNil(t, GetClaims(ctx))
		}
	})
	t.Run("SSO", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&types.Claims{Claims: jwt.Claims{Subject: "my-sub"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(false)
		g, err := NewGatekeeper(Modes{SSO: true}, wfClient, kubeClient, nil, ssoIf, clientForAuthorization, "my-ns")
		if assert.NoError(t, err) {
			ctx, err := g.Context(x("Bearer v2:whatever"))
			if assert.NoError(t, err) {
				assert.Equal(t, wfClient, GetWfClient(ctx))
				assert.Equal(t, kubeClient, GetKubeClient(ctx))
				if assert.NotNil(t, GetClaims(ctx)) {
					assert.Equal(t, "my-sub", GetClaims(ctx).Subject)
				}
			}
		}
	})
	hook := &test.Hook{}
	log.AddHook(hook)
	defer log.StandardLogger().ReplaceHooks(nil)
	t.Run("SSO+RBAC,precedence=1", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&types.Claims{Groups: []string{"my-group", "other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, wfClient, kubeClient, nil, ssoIf, clientForAuthorization, "my-ns")
		if assert.NoError(t, err) {
			ctx, err := g.Context(x("Bearer v2:whatever"))
			if assert.NoError(t, err) {
				assert.NotEqual(t, wfClient, GetWfClient(ctx))
				assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
				if assert.NotNil(t, GetClaims(ctx)) {
					assert.Equal(t, []string{"my-group", "other-group"}, GetClaims(ctx).Groups)
				}
				assert.Equal(t, "my-sa", hook.LastEntry().Data["serviceAccount"])
			}
		}
	})
	t.Run("SSO+RBAC,precedence=0", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&types.Claims{Groups: []string{"other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, wfClient, kubeClient, nil, ssoIf, clientForAuthorization, "my-ns")
		if assert.NoError(t, err) {
			_, err := g.Context(x("Bearer v2:whatever"))
			if assert.NoError(t, err) {
				assert.Equal(t, "my-other-sa", hook.LastEntry().Data["serviceAccount"])
			}
		}
	})
	t.Run("SSO+RBAC,denied", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&types.Claims{}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, wfClient, kubeClient, nil, ssoIf, clientForAuthorization, "my-ns")
		if assert.NoError(t, err) {
			_, err := g.Context(x("Bearer v2:whatever"))
			assert.EqualError(t, err, "rpc error: code = PermissionDenied desc = not allowed")
		}
	})
}

func x(authorization string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": authorization}))
}

func TestGetClaimSet(t *testing.T) {
	// we should be able to get nil claim set
	assert.Nil(t, GetClaims(context.TODO()))
}
