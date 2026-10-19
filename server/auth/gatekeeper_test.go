package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/config"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	headerauth "github.com/argoproj/argo-workflows/v4/server/auth/header"
	headermocks "github.com/argoproj/argo-workflows/v4/server/auth/header/mocks"
	ssomocks "github.com/argoproj/argo-workflows/v4/server/auth/sso/mocks"
	authTypes "github.com/argoproj/argo-workflows/v4/server/auth/types"
	"github.com/argoproj/argo-workflows/v4/server/cache"
	servertypes "github.com/argoproj/argo-workflows/v4/server/types"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestServer_GetWFClient(t *testing.T) {
	// prevent using local KUBECONFIG - which will fail on CI
	t.Setenv("KUBECONFIG", "/dev/null")
	wfClient := fakewfclientset.NewClientset()
	kubeClient := kubefake.NewClientset(
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
				Name: "user1-only-sa", Namespace: "user1-ns",
				Annotations: map[string]string{
					common.AnnotationKeyRBACRule:           "'user1-only-group' in groups",
					common.AnnotationKeyRBACRulePrecedence: "0",
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
	ctx := logging.TestContext(t.Context())
	resourceCache.Run(ctx.Done())
	var clientForAuthorization ClientForAuthorization = func(authorization string, config *rest.Config) (*rest.Config, *servertypes.Clients, error) {
		return &rest.Config{}, &servertypes.Clients{Workflow: &fakewfclientset.Clientset{}, Kubernetes: &kubefake.Clientset{}}, nil
	}
	clients := &servertypes.Clients{Workflow: wfClient, Kubernetes: kubeClient}
	t.Run("None", func(t *testing.T) {
		_, err := NewGatekeeper(Modes{}, clients, nil, nil, nil, clientForAuthorization, "", "", true, resourceCache, time.Minute)
		require.Error(t, err)
	})
	t.Run("Invalid", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Client: true}, clients, nil, nil, nil, clientForAuthorization, "", "", true, resourceCache, time.Minute)
		require.NoError(t, err)
		_, err = g.Context(x(logging.TestContext(t.Context()), "invalid"))
		require.Error(t, err)
	})
	t.Run("NotAllowed", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{SSO: true}, clients, nil, nil, nil, clientForAuthorization, "", "", true, resourceCache, time.Minute)
		require.NoError(t, err)
		_, err = g.Context(x(logging.TestContext(t.Context()), "Bearer "))
		require.Error(t, err)
	})
	t.Run("Client", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Client: true}, clients, &rest.Config{Username: "my-username"}, nil, nil, clientForAuthorization, "", "", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.Context(x(logging.TestContext(t.Context()), "Bearer "))
		require.NoError(t, err)
		assert.NotEqual(t, wfClient, GetWfClient(ctx))
		assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
		assert.Nil(t, GetClaims(ctx))
	})
	// reviewingClientForAuthorization returns a ClientForAuthorization whose kube client answers
	// SelfSubjectReviews with reviewErr, and a counter of how many reviews were attempted.
	reviewingClientForAuthorization := func(reviewErr error) (ClientForAuthorization, *int) {
		reviews := 0
		reviewingKubeClient := kubefake.NewClientset()
		reviewingKubeClient.PrependReactor("create", "selfsubjectreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
			reviews++
			return true, nil, reviewErr
		})
		return func(string, *rest.Config) (*rest.Config, *servertypes.Clients, error) {
			return &rest.Config{}, &servertypes.Clients{Workflow: &fakewfclientset.Clientset{}, Kubernetes: reviewingKubeClient}, nil
		}, &reviews
	}
	t.Run("ClientTokenRejected", func(t *testing.T) {
		rejecting, reviews := reviewingClientForAuthorization(apierrors.NewUnauthorized("Unauthorized"))
		g, err := NewGatekeeper(Modes{Client: true}, clients, &rest.Config{Username: "my-username"}, nil, nil, rejecting, "", "", true, resourceCache, time.Minute)
		require.NoError(t, err)
		for range 2 {
			_, err = g.Context(x(logging.TestContext(t.Context()), "Bearer fake"))
			require.Error(t, err)
			assert.Equal(t, codes.Unauthenticated, status.Code(err))
		}
		assert.Equal(t, 2, *reviews, "failed reviews must not be cached")
	})
	t.Run("ClientTokenReviewCached", func(t *testing.T) {
		accepting, reviews := reviewingClientForAuthorization(nil)
		g, err := NewGatekeeper(Modes{Client: true}, clients, &rest.Config{Username: "my-username"}, nil, nil, accepting, "", "", true, resourceCache, time.Minute)
		require.NoError(t, err)
		for range 2 {
			_, err = g.Context(x(logging.TestContext(t.Context()), "Bearer one"))
			require.NoError(t, err)
		}
		assert.Equal(t, 1, *reviews, "a successful review is cached for the same token")
		_, err = g.Context(x(logging.TestContext(t.Context()), "Bearer two"))
		require.NoError(t, err)
		assert.Equal(t, 2, *reviews, "a different token is reviewed")
	})
	t.Run("ClientTokenReviewCacheDisabled", func(t *testing.T) {
		accepting, reviews := reviewingClientForAuthorization(nil)
		g, err := NewGatekeeper(Modes{Client: true}, clients, &rest.Config{Username: "my-username"}, nil, nil, accepting, "", "", true, resourceCache, 0)
		require.NoError(t, err)
		for range 2 {
			_, err = g.Context(x(logging.TestContext(t.Context()), "Bearer one"))
			require.NoError(t, err)
		}
		assert.Equal(t, 2, *reviews, "a zero TTL disables the cache")
	})
	t.Run("Server", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Server: true}, clients, &rest.Config{Username: "my-username"}, nil, nil, clientForAuthorization, "", "", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.Context(x(logging.TestContext(t.Context()), ""))
		require.NoError(t, err)
		assert.Equal(t, wfClient, GetWfClient(ctx))
		assert.Equal(t, kubeClient, GetKubeClient(ctx))
		assert.NotNil(t, GetClaims(ctx))
	})
	t.Run("SSO", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "my-sub"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(false)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.Context(x(logging.TestContext(t.Context()), "Bearer v2:whatever"))
		require.NoError(t, err)
		assert.Equal(t, wfClient, GetWfClient(ctx))
		assert.Equal(t, kubeClient, GetKubeClient(ctx))
		require.NotNil(t, GetClaims(ctx))
		assert.Equal(t, "my-sub", GetClaims(ctx).Subject)
	})

	t.Run("Header", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "my-sub"}}, nil)
		headerIf.On("IsRBACEnabled").Return(false)
		g, err := NewGatekeeper(Modes{Header: true}, clients, &rest.Config{Username: "my-username"}, nil, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("x-forwarded-user", "pradeep"))
		ctx, err = g.Context(ctx)
		require.NoError(t, err)
		assert.Equal(t, wfClient, GetWfClient(ctx))
		assert.Equal(t, kubeClient, GetKubeClient(ctx))
		require.NotNil(t, GetClaims(ctx))
		assert.Equal(t, "my-sub", GetClaims(ctx).Subject)
	})

	t.Run("Header+RBAC,precedence=1", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(&authTypes.Claims{Groups: []string{"my-group", "other-group"}}, nil)
		headerIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{Header: true}, clients, &rest.Config{Username: "my-username"}, nil, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("x-forwarded-user", "pradeep"))
		ctx, err = g.Context(ctx)
		require.NoError(t, err)
		claims := GetClaims(ctx)
		require.NotNil(t, claims)
		assert.Equal(t, []string{"my-group", "other-group"}, claims.Groups)
		assert.Equal(t, "my-sa", claims.ServiceAccountName)
		assert.Equal(t, "my-ns", claims.ServiceAccountNamespace)
	})

	t.Run("Header+RBAC,denied", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "my-sub"}, Groups: []string{"unknown-group"}}, nil)
		headerIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{Header: true}, clients, &rest.Config{Username: "my-username"}, nil, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("x-forwarded-user", "pradeep"))
		_, err = g.Context(ctx)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("Authorization takes precedence over Header", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "header-user"}}, nil)
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "sso-user"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(false)
		g, err := NewGatekeeper(Modes{SSO: true, Header: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("authorization", "Bearer v2:whatever", "x-forwarded-user", "header-user"))
		ctx, err = g.Context(ctx)
		require.NoError(t, err)
		require.NotNil(t, GetClaims(ctx))
		assert.Equal(t, "sso-user", GetClaims(ctx).Subject)
		headerIf.AssertNotCalled(t, "Authorize", mock.Anything)
	})
	t.Run("Header+Server, Header no credentials falls through to Server", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(nil, authTypes.ErrNoCredentials)
		g, err := NewGatekeeper(Modes{Header: true, Server: true}, clients, &rest.Config{Username: "my-username"}, nil, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("x-forwarded-user", ""))
		ctx, err = g.Context(ctx)
		require.NoError(t, err)
		assert.Equal(t, wfClient, GetWfClient(ctx))
		assert.Equal(t, kubeClient, GetKubeClient(ctx))
		assert.NotNil(t, GetClaims(ctx))
	})

	t.Run("Header+Server, missing identity header falls through to Server", func(t *testing.T) {
		headerIf, err := headerauth.New(t.Context(), config.HeaderConfig{
			Subject: config.ClaimSource{
				Header: "X-Forwarded-User",
			},
		}, kubefake.NewClientset().CoreV1().Secrets("my-ns"), true)
		require.NoError(t, err)

		g, err := NewGatekeeper(Modes{Header: true, Server: true}, clients, &rest.Config{Username: "my-username"}, nil, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)

		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.MD{})
		ctx, err = g.Context(ctx)

		require.NoError(t, err)
		assert.Equal(t, wfClient, GetWfClient(ctx))
		assert.Equal(t, kubeClient, GetKubeClient(ctx))
		assert.NotNil(t, GetClaims(ctx))
	})
	t.Run("Authorization failure does not fall back to Header", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "header-user"}}, nil)
		headerIf.On("IsRBACEnabled").Return(false)
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything).Return(nil, errors.New("invalid credentials"))
		g, err := NewGatekeeper(Modes{SSO: true, Header: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("authorization", "Bearer v2:invalid", "x-forwarded-user", "header-user"))
		_, err = g.Context(ctx)
		ssoIf.AssertExpectations(t)
		headerIf.AssertNotCalled(t, "Authorize", mock.Anything)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
	t.Run("Header takes precedence over cookie", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "header-user"}}, nil)
		headerIf.On("IsRBACEnabled").Return(false)
		ssoIf := &ssomocks.Interface{}
		g, err := NewGatekeeper(Modes{SSO: true, Header: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("cookie", "authorization=Bearer v2:invalid"))
		ctx, err = g.Context(ctx)
		require.NoError(t, err)
		require.NotNil(t, GetClaims(ctx))
		assert.Equal(t, "header-user", GetClaims(ctx).Subject)
		ssoIf.AssertNotCalled(t, "Authorize", mock.Anything)
	})
	t.Run("Header no credentials falls through to cookie", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(nil, authTypes.ErrNoCredentials)
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "cookie-user"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(false)
		g, err := NewGatekeeper(Modes{SSO: true, Header: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("cookie", "authorization=Bearer v2:valid"))
		ctx, err = g.Context(ctx)
		require.NoError(t, err)
		require.NotNil(t, GetClaims(ctx))
		assert.Equal(t, "cookie-user", GetClaims(ctx).Subject)
		ssoIf.AssertExpectations(t)
	})
	t.Run("Header failure does not fall back to cookie or Server", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(nil, errors.New("invalid header credentials"))
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "cookie-user"}}, nil)
		g, err := NewGatekeeper(Modes{SSO: true, Header: true, Server: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("cookie", "authorization=Bearer v2:valid"))
		_, err = g.Context(ctx)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		ssoIf.AssertNotCalled(t, "Authorize", mock.Anything)
	})

	t.Run("Unrecognised Authorization with SSO+Header is rejected, not header identity", func(t *testing.T) {
		// A k8s bearer token (no sso prefix) when client mode is off.
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "header-user"}}, nil)
		ssoIf := &ssomocks.Interface{}
		g, err := NewGatekeeper(Modes{SSO: true, Header: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("authorization", "Bearer k8s-token", "x-forwarded-user", "header-user"))
		_, err = g.Context(ctx)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		ssoIf.AssertNotCalled(t, "Authorize", mock.Anything, mock.Anything)
		headerIf.AssertNotCalled(t, "Authorize", mock.Anything)
	})

	t.Run("Unrecognised Authorization with Header+Server runs as server, not header identity", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		g, err := NewGatekeeper(Modes{Header: true, Server: true}, clients, &rest.Config{Username: "my-username"}, nil, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("authorization", "garbage", "x-forwarded-user", "header-user"))
		ctx, err = g.Context(ctx)
		require.NoError(t, err)
		assert.Equal(t, wfClient, GetWfClient(ctx))
		assert.Equal(t, kubeClient, GetKubeClient(ctx))
		headerIf.AssertNotCalled(t, "Authorize", mock.Anything)
	})

	for _, empty := range []string{"", "   "} {
		t.Run("Empty Authorization "+`"`+empty+`"`+" falls through to Header", func(t *testing.T) {
			headerIf := &headermocks.Interface{}
			headerIf.On("Authorize", mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "header-user"}}, nil)
			headerIf.On("IsRBACEnabled").Return(false)
			g, err := NewGatekeeper(Modes{SSO: true, Header: true}, clients, &rest.Config{Username: "my-username"}, &ssomocks.Interface{}, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
			require.NoError(t, err)
			ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("authorization", empty, "x-forwarded-user", "header-user"))
			ctx, err = g.Context(ctx)
			require.NoError(t, err)
			require.NotNil(t, GetClaims(ctx))
			assert.Equal(t, "header-user", GetClaims(ctx).Subject)
		})
	}

	t.Run("SSO failure does not fall back to Server", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(nil, errors.New("invalid credentials"))
		g, err := NewGatekeeper(Modes{SSO: true, Server: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		_, err = g.Context(x(logging.TestContext(t.Context()), "Bearer v2:invalid"))
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("Client token rejected does not fall back to Header", func(t *testing.T) {
		rejecting, _ := reviewingClientForAuthorization(apierrors.NewUnauthorized("Unauthorized"))
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "header-user"}}, nil)
		g, err := NewGatekeeper(Modes{Client: true, Header: true}, clients, &rest.Config{Username: "my-username"}, nil, headerIf, rejecting, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("authorization", "Bearer fake", "x-forwarded-user", "header-user"))
		_, err = g.Context(ctx)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		headerIf.AssertNotCalled(t, "Authorize", mock.Anything)
	})

	t.Run("Cookie SSO failure does not fall back to Server", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(nil, authTypes.ErrNoCredentials)
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(nil, errors.New("invalid credentials"))
		g, err := NewGatekeeper(Modes{SSO: true, Header: true, Server: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("cookie", "authorization=Bearer v2:invalid"))
		_, err = g.Context(ctx)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("Unrecognised cookie credential runs as server", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{SSO: true, Server: true}, clients, &rest.Config{Username: "my-username"}, &ssomocks.Interface{}, nil, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("cookie", "authorization=Bearer k8s-token"))
		ctx, err = g.Context(ctx)
		require.NoError(t, err)
		assert.Equal(t, wfClient, GetWfClient(ctx))
		assert.Equal(t, kubeClient, GetKubeClient(ctx))
	})
	t.Run("Server only, Bearer token runs as server", func(t *testing.T) {
		g, err := NewGatekeeper(Modes{Server: true}, clients, &rest.Config{Username: "my-username"}, nil, nil, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.Context(x(logging.TestContext(t.Context()), "Bearer x"))
		require.NoError(t, err)
		assert.Equal(t, wfClient, GetWfClient(ctx))
		assert.Equal(t, kubeClient, GetKubeClient(ctx))
	})

	t.Run("Client+SSO+Server, Bearer v2: goes to SSO not client", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&authTypes.Claims{Claims: jwt.Claims{Subject: "sso-user"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(false)
		clientCalled := false
		spying := func(string, *rest.Config) (*rest.Config, *servertypes.Clients, error) {
			clientCalled = true
			return &rest.Config{}, &servertypes.Clients{Workflow: &fakewfclientset.Clientset{}, Kubernetes: &kubefake.Clientset{}}, nil
		}
		g, err := NewGatekeeper(Modes{Client: true, SSO: true, Server: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, spying, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.Context(x(logging.TestContext(t.Context()), "Bearer v2:whatever"))
		require.NoError(t, err)
		require.NotNil(t, GetClaims(ctx))
		assert.Equal(t, "sso-user", GetClaims(ctx).Subject)
		assert.False(t, clientCalled, "a v2: token must go to SSO, not client auth")
	})

	t.Run("Client+SSO+Server, plain Bearer goes to client not SSO", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		g, err := NewGatekeeper(Modes{Client: true, SSO: true, Server: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.Context(x(logging.TestContext(t.Context()), "Bearer k8s-token"))
		require.NoError(t, err)
		assert.NotEqual(t, wfClient, GetWfClient(ctx))
		ssoIf.AssertNotCalled(t, "Authorize", mock.Anything, mock.Anything)
	})

	t.Run("Header+Server, header error does not fall back to Server", func(t *testing.T) {
		headerIf := &headermocks.Interface{}
		headerIf.On("Authorize", mock.Anything).Return(nil, errors.New("invalid header credentials"))
		g, err := NewGatekeeper(Modes{Header: true, Server: true}, clients, &rest.Config{Username: "my-username"}, nil, headerIf, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(logging.TestContext(t.Context()), metadata.Pairs("x-forwarded-user", "someone"))
		_, err = g.Context(ctx)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
	t.Run("SSO+RBAC,precedence=1", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&authTypes.Claims{Groups: []string{"my-group", "other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.Context(x(logging.TestContext(t.Context()), "Bearer v2:whatever"))
		require.NoError(t, err)
		assert.NotEqual(t, clients, GetWfClient(ctx))
		assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
		claims := GetClaims(ctx)
		require.NotNil(t, claims)
		assert.Equal(t, []string{"my-group", "other-group"}, claims.Groups)
		assert.Equal(t, "my-sa", claims.ServiceAccountName)
		assert.Equal(t, "my-ns", claims.ServiceAccountNamespace)
	})
	t.Run("SSO+RBAC, Namespace delegation ON, precedence=2, Delegated", func(t *testing.T) {
		t.Setenv("SSO_DELEGATE_RBAC_TO_NAMESPACE", "true")
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&authTypes.Claims{Groups: []string{"my-group", "other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, clientForAuthorization, "my-ns", "my-ns", false, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.ContextWithRequest(x(logging.TestContext(t.Context()), "Bearer v2:whatever"), servertypes.NamespaceHolder("user1-ns"))
		require.NoError(t, err)
		assert.NotEqual(t, clients, GetWfClient(ctx))
		assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
		claims := GetClaims(ctx)
		require.NotNil(t, claims)
		assert.Equal(t, []string{"my-group", "other-group"}, claims.Groups)
		assert.Equal(t, "user1-sa", claims.ServiceAccountName)
		assert.Equal(t, "user1-ns", claims.ServiceAccountNamespace)
	})
	t.Run("SSO+RBAC, Namespace delegation OFF, precedence=2, Not Delegated", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&authTypes.Claims{Groups: []string{"my-group", "other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.ContextWithRequest(x(logging.TestContext(t.Context()), "Bearer v2:whatever"), servertypes.NamespaceHolder("user1-ns"))
		require.NoError(t, err)
		assert.NotEqual(t, clients, GetWfClient(ctx))
		assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
		claims := GetClaims(ctx)
		require.NotNil(t, claims)
		assert.Equal(t, []string{"my-group", "other-group"}, claims.Groups)
		assert.Equal(t, "my-sa", claims.ServiceAccountName)
		assert.Equal(t, "my-ns", claims.ServiceAccountNamespace)
	})
	t.Run("SSO+RBAC, Namespace delegation ON, precedence=0, Not delegated", func(t *testing.T) {
		t.Setenv("SSO_DELEGATE_RBAC_TO_NAMESPACE", "true")
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&authTypes.Claims{Groups: []string{"my-group", "other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, clientForAuthorization, "my-ns", "my-ns", false, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.ContextWithRequest(x(logging.TestContext(t.Context()), "Bearer v2:whatever"), servertypes.NamespaceHolder("user2-ns"))
		require.NoError(t, err)
		assert.NotEqual(t, clients, GetWfClient(ctx))
		assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
		claims := GetClaims(ctx)
		require.NotNil(t, claims)
		assert.Equal(t, []string{"my-group", "other-group"}, claims.Groups)
		assert.Equal(t, "my-sa", claims.ServiceAccountName)
		assert.Equal(t, "my-ns", claims.ServiceAccountNamespace)
	})
	t.Run("SSO+RBAC, Namespace delegation ON, precedence=1, Not delegated", func(t *testing.T) {
		t.Setenv("SSO_DELEGATE_RBAC_TO_NAMESPACE", "true")
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&authTypes.Claims{Groups: []string{"my-group", "other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, clientForAuthorization, "my-ns", "my-ns", false, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.ContextWithRequest(x(logging.TestContext(t.Context()), "Bearer v2:whatever"), servertypes.NamespaceHolder("user3-ns"))
		require.NoError(t, err)
		assert.NotEqual(t, clients, GetWfClient(ctx))
		assert.NotEqual(t, kubeClient, GetKubeClient(ctx))
		claims := GetClaims(ctx)
		require.NotNil(t, claims)
		assert.Equal(t, []string{"my-group", "other-group"}, claims.Groups)
		assert.Equal(t, "my-sa", claims.ServiceAccountName)
		assert.Equal(t, "my-ns", claims.ServiceAccountNamespace)
	})
	t.Run("SSO+RBAC, Namespace delegation ON, no ssoNamespace match, Delegated", func(t *testing.T) {
		// Regression test: claims match a namespace-level rule but no
		// ssoNamespace rule. Previously this caused a nil pointer panic
		// because precedence(loginAccount) was called on nil.
		t.Setenv("SSO_DELEGATE_RBAC_TO_NAMESPACE", "true")
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&authTypes.Claims{Groups: []string{"user1-only-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, clientForAuthorization, "my-ns", "my-ns", false, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.ContextWithRequest(x(logging.TestContext(t.Context()), "Bearer v2:whatever"), servertypes.NamespaceHolder("user1-ns"))
		require.NoError(t, err)
		claims := GetClaims(ctx)
		require.NotNil(t, claims)
		assert.Equal(t, "user1-only-sa", claims.ServiceAccountName)
		assert.Equal(t, "user1-ns", claims.ServiceAccountNamespace)
	})
	t.Run("SSO+RBAC,precedence=0", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&authTypes.Claims{Groups: []string{"other-group"}}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		ctx, err := g.Context(x(logging.TestContext(t.Context()), "Bearer v2:whatever"))
		require.NoError(t, err)
		assert.Equal(t, "my-other-sa", GetClaims(ctx).ServiceAccountName)
	})
	t.Run("SSO+RBAC,denied", func(t *testing.T) {
		ssoIf := &ssomocks.Interface{}
		ssoIf.On("Authorize", mock.Anything, mock.Anything).Return(&authTypes.Claims{}, nil)
		ssoIf.On("IsRBACEnabled").Return(true)
		g, err := NewGatekeeper(Modes{SSO: true}, clients, &rest.Config{Username: "my-username"}, ssoIf, nil, clientForAuthorization, "my-ns", "my-ns", true, resourceCache, time.Minute)
		require.NoError(t, err)
		_, err = g.Context(x(logging.TestContext(t.Context()), "Bearer v2:whatever"))
		require.EqualError(t, err, "rpc error: code = PermissionDenied desc = not allowed")
	})
}

func x(ctx context.Context, authorization string) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.New(map[string]string{"authorization": authorization}))
}

func TestGetClaimSet(t *testing.T) {
	assert.Nil(t, GetClaims(logging.TestContext(t.Context())))
}
