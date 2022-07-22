package auth

import (
	"context"
	"os"
	"testing"

	"github.com/go-jose/go-jose/v3/jwt"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	fakewfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	ssomocks "github.com/argoproj/argo-workflows/v3/server/auth/sso/mocks"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
	"github.com/argoproj/argo-workflows/v3/server/cache"
	servertypes "github.com/argoproj/argo-workflows/v3/server/types"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
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
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user1-sa", Namespace: "user1-ns",
				Annotations: map[string]string{
					common.AnnotationKeyRBACRule:           "'my-group' in groups",
					common.AnnotationKeyRBACRulePrecedence: "2",
				},
			},
			Secrets: []corev1.ObjectReference{{Name: "user-secret"}},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user2-sa", Namespace: "user2-ns",
				Annotations: map[string]string{
					common.AnnotationKeyRBACRule:           "'my-group' in groups",
					common.AnnotationKeyRBACRulePrecedence: "0",
				},
			},
			Secrets: []corev1.ObjectReference{{Name: "user-secret"}},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user3-sa", Namespace: "user3-ns",
				Annotations: map[string]string{
					common.AnnotationKeyRBACRule:           "'my-group' in groups",
					common.AnnotationKeyRBACRulePrecedence: "1",
				},
			},
			Secrets: []corev1.ObjectReference{{Name: "user-secret"}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "my-secret", Namespace: "my-ns"},
			Data: map[string][]byte{
				"token": {},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "user-secret", Namespace: "user1-ns"},
			Data: map[string][]byte{
				"token": {},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "user-secret", Namespace: "user2-ns"},
			Data: map[string][]byte{
				"token": {},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "user-secret", Namespace: "user3-ns"},
			Data: map[string][]byte{
				"token": {},
			},
		},
	)
	resourceCache := cache.NewResourceCache(kubeClient, corev1.NamespaceAll)
	resourceCache.Run(context.TODO().Done())
	var clientForAuthorization ClientForAuthorization = func(authorization string) (*rest.Config, *servertypes.Clients, error) {
		return &rest.Config{}, &servertypes.Clients{Workflow: &fakewfclientset.Clientset{}, Kubernetes: &kubefake.Clientset{}}, nil
	}
	clients := &servertypes.Clients{Workflow: wfClient, Kubernetes: kubeClient}
	t.Run("None", func(t *testing.T) {
		_, err := NewGatekeeper(Modes{}, clients, nil, nil, clientForAuthorization, "", "", true, resourceCache)
		assert.Error(t, err)
	})
	t.Run("Invalid", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Client: true}, clients, nil, nil, clientForAuthorization, "", "", true, resourceCache)
		if assert.NoError(t, err) {
			_, err := g.Context(x("invalid"))
			assert.Error(t, err)
		}
	})
	t.Run("NotAllowed", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{SSO: true}, clients, nil, nil, clientForAuthorization, "", "", true, resourceCache)
		if assert.NoError(t, err) {
			_, err := g.Context(x("Bearer "))
			assert.Error(t, err)
		}
	})
	t.Run("Client", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Client: true}, clients, &rest.Config{Username: "my-username"}, nil, clientForAuthorization, "", "", true, resourceCache)
		assert.NoError(t, err)
		ctx, err := g.Context(x("Bearer "))
		if assert.NoError(t, err) {
			assert.NotEqual(t, wfClient, GetWfClient(ctx))
			assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
			assert.Nil(t, GetClaims(ctx))
		}
	})
	t.Run("Server", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Server: true}, clients, &rest.Config{Username: "my-username"}, nil, clientForAuthorization, "", "", true, resourceCache)
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
		g, err := NewGatekeeper(Modes{SSO: true}, clients, nil, ssoIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache)
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
		g, err := NewGatekeeper(Modes{SSO: true}, clients, nil, ssoIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache)
		if assert.NoError(t, err) {
			ctx, err := g.Context(x("Bearer v2:whatever"))
			if assert.NoError(t, err) {
				assert.NotEqual(t, clients, GetWfClient(ctx))
				assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
				claims := GetClaims(ctx)
				if assert.NotNil(t, claims) {
					assert.Equal(t, []string{"my-group", "other-group"}, claims.Groups)
					assert.Equal(t, "my-sa", claims.ServiceAccountName)
					assert.Equal(t, "my-ns", claims.ServiceAccountNamespace)
				}
				assert.Equal(t, "my-sa", hook.LastEntry().Data["serviceAccount"])
			}
		}
	})
	t.Run("SSO+RBAC, Namespace delegation ON, precedence=2, Delegated", func(t *testing.T) {
		os.Setenv("SSO_DELEGATE_RBAC_TO_NAMESPACE", "true")
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&types.Claims{Groups: []string{"my-group", "other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, nil, ssoIf, clientForAuthorization, "my-ns", "my-ns", false, resourceCache)
		if assert.NoError(t, err) {
			ctx, err := g.ContextWithRequest(x("Bearer v2:whatever"), servertypes.NamespaceHolder("user1-ns"))
			if assert.NoError(t, err) {
				assert.NotEqual(t, clients, GetWfClient(ctx))
				assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
				claims := GetClaims(ctx)
				if assert.NotNil(t, claims) {
					assert.Equal(t, []string{"my-group", "other-group"}, claims.Groups)
					assert.Equal(t, "user1-sa", claims.ServiceAccountName)
					assert.Equal(t, "user1-ns", claims.ServiceAccountNamespace)
				}
				assert.Equal(t, "user1-sa", hook.LastEntry().Data["serviceAccount"])
			}
		}
		os.Unsetenv("SSO_DELEGATE_RBAC_TO_NAMESPACE")
	})
	t.Run("SSO+RBAC, Namespace delegation OFF, precedence=2, Not Delegated", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&types.Claims{Groups: []string{"my-group", "other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, nil, ssoIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache)
		if assert.NoError(t, err) {
			ctx, err := g.ContextWithRequest(x("Bearer v2:whatever"), servertypes.NamespaceHolder("user1-ns"))
			if assert.NoError(t, err) {
				assert.NotEqual(t, clients, GetWfClient(ctx))
				assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
				claims := GetClaims(ctx)
				if assert.NotNil(t, claims) {
					assert.Equal(t, []string{"my-group", "other-group"}, claims.Groups)
					assert.Equal(t, "my-sa", claims.ServiceAccountName)
					assert.Equal(t, "my-ns", claims.ServiceAccountNamespace)
				}
				assert.Equal(t, "my-sa", hook.LastEntry().Data["serviceAccount"])
			}
		}
	})
	t.Run("SSO+RBAC, Namespace delegation ON, precedence=0, Not delegated", func(t *testing.T) {
		os.Setenv("SSO_DELEGATE_RBAC_TO_NAMESPACE", "true")
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&types.Claims{Groups: []string{"my-group", "other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, nil, ssoIf, clientForAuthorization, "my-ns", "my-ns", false, resourceCache)
		if assert.NoError(t, err) {
			ctx, err := g.ContextWithRequest(x("Bearer v2:whatever"), servertypes.NamespaceHolder("user2-ns"))
			if assert.NoError(t, err) {
				assert.NotEqual(t, clients, GetWfClient(ctx))
				assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
				claims := GetClaims(ctx)
				if assert.NotNil(t, claims) {
					assert.Equal(t, []string{"my-group", "other-group"}, claims.Groups)
					assert.Equal(t, "my-sa", claims.ServiceAccountName)
					assert.Equal(t, "my-ns", claims.ServiceAccountNamespace)
				}
				assert.Equal(t, "my-sa", hook.LastEntry().Data["serviceAccount"])
			}
		}
		os.Unsetenv("SSO_DELEGATE_RBAC_TO_NAMESPACE")
	})
	t.Run("SSO+RBAC, Namespace delegation ON, precedence=1, Not delegated", func(t *testing.T) {
		os.Setenv("SSO_DELEGATE_RBAC_TO_NAMESPACE", "true")
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&types.Claims{Groups: []string{"my-group", "other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, nil, ssoIf, clientForAuthorization, "my-ns", "my-ns", false, resourceCache)
		if assert.NoError(t, err) {
			ctx, err := g.ContextWithRequest(x("Bearer v2:whatever"), servertypes.NamespaceHolder("user3-ns"))
			if assert.NoError(t, err) {
				assert.NotEqual(t, clients, GetWfClient(ctx))
				assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
				claims := GetClaims(ctx)
				if assert.NotNil(t, claims) {
					assert.Equal(t, []string{"my-group", "other-group"}, claims.Groups)
					assert.Equal(t, "my-sa", claims.ServiceAccountName)
					assert.Equal(t, "my-ns", claims.ServiceAccountNamespace)
				}
				assert.Equal(t, "my-sa", hook.LastEntry().Data["serviceAccount"])
			}
		}
		os.Unsetenv("SSO_DELEGATE_RBAC_TO_NAMESPACE")
	})
	t.Run("SSO+RBAC,precedence=0", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&types.Claims{Groups: []string{"other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, nil, ssoIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache)
		if assert.NoError(t, err) {
			ctx, err := g.Context(x("Bearer v2:whatever"))
			if assert.NoError(t, err) {
				assert.Equal(t, "my-other-sa", hook.LastEntry().Data["serviceAccount"])
				assert.Equal(t, "my-other-sa", GetClaims(ctx).ServiceAccountName)
			}
		}
	})
	t.Run("SSO+RBAC,denied", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&types.Claims{}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, nil, ssoIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache)
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
