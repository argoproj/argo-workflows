package header

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-workflows/v4/config"
)

func TestAuthorize(t *testing.T) {
	tests := []struct {
		name string

		cfg config.HeaderConfig
		md  metadata.MD

		trustUnauthenticated bool

		issuer  string
		subject string
		email   string
		groups  []string
	}{
		{
			name: "static values",

			cfg: config.HeaderConfig{
				Issuer: config.ClaimSource{
					Value: "oauth2-proxy",
				},
				Subject: config.ClaimSource{
					Value: "pradeep",
				},
			},

			md: metadata.MD{},

			trustUnauthenticated: true,

			issuer:  "oauth2-proxy",
			subject: "pradeep",
		},

		{
			name: "header values",

			cfg: config.HeaderConfig{
				Subject: config.ClaimSource{
					Header: "X-Forwarded-User",
				},
				Email: config.ClaimSource{
					Header: "X-Forwarded-Email",
				},
			},

			md: metadata.Pairs(
				"x-forwarded-user", "pradeep",
				"x-forwarded-email", "abc@test.com",
			),

			trustUnauthenticated: true,

			subject: "pradeep",
			email:   "abc@test.com",
		},

		{
			name: "groups",

			cfg: config.HeaderConfig{
				Subject: config.ClaimSource{
					Header: "X-Forwarded-User",
				},
				Groups: config.GroupClaimSource{
					ClaimSource: config.ClaimSource{
						Header: "X-Forwarded-Groups",
					},
				},
			},

			md: metadata.Pairs(
				"x-forwarded-user", "pradeep",
				"x-forwarded-groups", "admin,developer,argo",
			),

			trustUnauthenticated: true,

			subject: "pradeep",
			groups: []string{
				"admin",
				"developer",
				"argo",
			},
		},

		{
			name: "multiple values for same header",

			cfg: config.HeaderConfig{
				Subject: config.ClaimSource{
					Header: "X-Forwarded-User",
				},
			},

			md: metadata.Pairs(
				"x-forwarded-user", "pradeep",
				"x-forwarded-user", "admin",
			),

			trustUnauthenticated: true,

			subject: "pradeep,admin",
		},

		{
			name: "groups with whitespace and empty entries",

			cfg: config.HeaderConfig{
				Subject: config.ClaimSource{
					Header: "X-Forwarded-User",
				},
				Groups: config.GroupClaimSource{
					ClaimSource: config.ClaimSource{
						Header: "X-Forwarded-Groups",
					},
				},
			},

			md: metadata.Pairs(
				"x-forwarded-user", "pradeep",
				"x-forwarded-groups", "admin, developer,,argo, ",
			),

			trustUnauthenticated: true,

			subject: "pradeep",
			groups: []string{
				"admin",
				" developer",
				"",
				"argo",
				" ",
			},
		},

		{
			name: "shared secret authentication",

			cfg: config.HeaderConfig{
				SharedSecret: &config.SharedSecretHeader{
					Header: "X-Proxy-Auth",
					RequiredValue: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "argo-header-auth",
						},
						Key: "proxy-secret",
					},
				},
				Subject: config.ClaimSource{
					Header: "X-Forwarded-User",
				},
			},

			md: metadata.Pairs(
				"x-proxy-auth", "secret",
				"x-forwarded-user", "pradeep",
			),

			subject: "pradeep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			if tt.cfg.SharedSecret != nil {
				_, err := clientset.CoreV1().
					Secrets("default").
					Create(
						context.Background(),
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name: "argo-header-auth",
							},
							Data: map[string][]byte{
								"proxy-secret": []byte("secret"),
							},
						},
						metav1.CreateOptions{},
					)
				require.NoError(t, err)
			}

			h, err := New(
				context.Background(),
				tt.cfg,
				clientset.CoreV1().Secrets("default"),
				tt.trustUnauthenticated,
			)
			require.NoError(t, err)

			claims, err := h.Authorize(tt.md)

			require.NoError(t, err)
			assert.Equal(t, tt.issuer, claims.Issuer)
			assert.Equal(t, tt.subject, claims.Subject)
			assert.Equal(t, tt.email, claims.Email)
			assert.Equal(t, tt.groups, claims.Groups)
		})
	}
}

func TestIsRBACEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.HeaderConfig
		want bool
	}{
		{
			name: "RBAC not configured",
			cfg:  config.HeaderConfig{},
			want: false,
		},
		{
			name: "RBAC disabled",
			cfg: config.HeaderConfig{
				RBAC: &config.RBACConfig{Enabled: false},
			},
			want: false,
		},
		{
			name: "RBAC enabled",
			cfg: config.HeaderConfig{
				RBAC: &config.RBACConfig{Enabled: true},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			h, err := New(
				context.Background(),
				tt.cfg,
				clientset.CoreV1().Secrets("default"),
				true,
			)
			require.NoError(t, err)

			assert.Equal(t, tt.want, h.IsRBACEnabled())
		})
	}
}

func TestAuthorizeMissingSubject(t *testing.T) {
	cfg := config.HeaderConfig{
		Subject: config.ClaimSource{
			Header: "X-Forwarded-User",
		},
	}

	clientset := fake.NewSimpleClientset()

	h, err := New(
		context.Background(),
		cfg,
		clientset.CoreV1().Secrets("default"),
		true,
	)
	require.NoError(t, err)

	claims, err := h.Authorize(metadata.MD{})

	assert.Nil(t, claims)
	assert.EqualError(t, err, "subject claim is empty")
}

func TestAuthenticateProxy(t *testing.T) {
	tests := []struct {
		name          string
		cfg           config.HeaderConfig
		md            metadata.MD
		expectedError string
	}{
		{
			name: "valid secret",
			cfg: config.HeaderConfig{
				SharedSecret: &config.SharedSecretHeader{
					Header: "X-Proxy-Auth",
					RequiredValue: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "argo-header-auth",
						},
						Key: "proxy-secret",
					},
				},
			},
			md: metadata.Pairs(
				"x-proxy-auth", "secret",
			),
		},
		{
			name: "invalid secret",
			cfg: config.HeaderConfig{
				SharedSecret: &config.SharedSecretHeader{
					Header: "X-Proxy-Auth",
					RequiredValue: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "argo-header-auth",
						},
						Key: "proxy-secret",
					},
				},
			},
			md: metadata.Pairs(
				"x-proxy-auth", "wrong-secret",
			),
			expectedError: "trusted proxy authentication failed",
		},
		{
			name: "missing authentication header",
			cfg: config.HeaderConfig{
				SharedSecret: &config.SharedSecretHeader{
					Header: "X-Proxy-Auth",
					RequiredValue: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "argo-header-auth",
						},
						Key: "proxy-secret",
					},
				},
			},
			md:            metadata.MD{},
			expectedError: "trusted proxy authentication header is missing",
		},
		{
			name: "multiple authentication headers",
			cfg: config.HeaderConfig{
				SharedSecret: &config.SharedSecretHeader{
					Header: "X-Proxy-Auth",
					RequiredValue: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "argo-header-auth",
						},
						Key: "proxy-secret",
					},
				},
			},
			md: metadata.Pairs(
				"x-proxy-auth", "wrong-secret",
				"x-proxy-auth", "secret",
			),
			expectedError: "trusted proxy authentication failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			_, err := clientset.CoreV1().
				Secrets("default").
				Create(
					context.Background(),
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "argo-header-auth",
						},
						Data: map[string][]byte{
							"proxy-secret": []byte("secret"),
						},
					},
					metav1.CreateOptions{},
				)
			require.NoError(t, err)

			h, err := New(
				context.Background(),
				tt.cfg,
				clientset.CoreV1().Secrets("default"),
				false,
			)
			require.NoError(t, err)

			err = h.(*header).authenticateProxy(tt.md)

			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAuthorizeProxyAuthentication(t *testing.T) {
	cfg := config.HeaderConfig{
		SharedSecret: &config.SharedSecretHeader{
			Header: "X-Proxy-Auth",
			RequiredValue: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "argo-header-auth",
				},
				Key: "proxy-secret",
			},
		},
		Subject: config.ClaimSource{
			Header: "X-Forwarded-User",
		},
	}

	t.Run("secure mode requires valid proxy secret", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()

		_, err := clientset.CoreV1().
			Secrets("default").
			Create(
				context.Background(),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "argo-header-auth",
					},
					Data: map[string][]byte{
						"proxy-secret": []byte("secret"),
					},
				},
				metav1.CreateOptions{},
			)
		require.NoError(t, err)

		h, err := New(
			context.Background(),
			cfg,
			clientset.CoreV1().Secrets("default"),
			false,
		)
		require.NoError(t, err)

		claims, err := h.Authorize(metadata.Pairs(
			"x-proxy-auth", "secret",
			"x-forwarded-user", "pradeep",
		))

		require.NoError(t, err)
		assert.Equal(t, "pradeep", claims.Subject)
	})

	t.Run("secure mode rejects invalid proxy secret", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()

		_, err := clientset.CoreV1().
			Secrets("default").
			Create(
				context.Background(),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "argo-header-auth",
					},
					Data: map[string][]byte{
						"proxy-secret": []byte("secret"),
					},
				},
				metav1.CreateOptions{},
			)
		require.NoError(t, err)

		h, err := New(
			context.Background(),
			cfg,
			clientset.CoreV1().Secrets("default"),
			false,
		)
		require.NoError(t, err)

		claims, err := h.Authorize(metadata.Pairs(
			"x-proxy-auth", "wrong-secret",
			"x-forwarded-user", "pradeep",
		))

		assert.Nil(t, claims)
		require.EqualError(t, err, "trusted proxy authentication failed")
	})

	t.Run("secure mode rejects missing proxy secret", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()

		_, err := clientset.CoreV1().
			Secrets("default").
			Create(
				context.Background(),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "argo-header-auth",
					},
					Data: map[string][]byte{
						"proxy-secret": []byte("secret"),
					},
				},
				metav1.CreateOptions{},
			)
		require.NoError(t, err)

		h, err := New(
			context.Background(),
			cfg,
			clientset.CoreV1().Secrets("default"),
			false,
		)
		require.NoError(t, err)

		claims, err := h.Authorize(metadata.Pairs(
			"x-forwarded-user", "pradeep",
		))

		assert.Nil(t, claims)
		require.EqualError(t, err, "trusted proxy authentication header is missing")
	})

	t.Run("insecure mode skips proxy authentication", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()

		h, err := New(
			context.Background(),
			cfg,
			clientset.CoreV1().Secrets("default"),
			true,
		)
		require.NoError(t, err)

		claims, err := h.Authorize(metadata.Pairs(
			"x-forwarded-user", "pradeep",
		))

		require.NoError(t, err)
		assert.Equal(t, "pradeep", claims.Subject)
	})
}

func TestGetHeaderSharedSecret(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.SharedSecretHeader
		secret        *corev1.Secret
		expected      string
		expectedError string
	}{
		{
			name: "valid secret",
			cfg: &config.SharedSecretHeader{
				Header: "X-Proxy-Auth",
				RequiredValue: corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "argo-header-auth",
					},
					Key: "proxy-secret",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "argo-header-auth",
				},
				Data: map[string][]byte{
					"proxy-secret": []byte("secret"),
				},
			},
			expected: "secret",
		},
		{
			name:          "shared secret not configured",
			cfg:           nil,
			expectedError: "shared secret authentication is not configured",
		},
		{
			name: "secret reference is empty",
			cfg: &config.SharedSecretHeader{
				Header: "X-Proxy-Auth",
			},
			expectedError: "shared secret reference is empty",
		},
		{
			name: "secret does not exist",
			cfg: &config.SharedSecretHeader{
				Header: "X-Proxy-Auth",
				RequiredValue: corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "missing-secret",
					},
					Key: "proxy-secret",
				},
			},
			expectedError: `failed to get shared secret: secrets "missing-secret" not found`,
		},
		{
			name: "shared secret header is empty",
			cfg: &config.SharedSecretHeader{
				RequiredValue: corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "argo-header-auth",
					},
					Key: "proxy-secret",
				},
			},
			expectedError: "shared secret header is empty",
		},
		{
			name: "secret value is empty",
			cfg: &config.SharedSecretHeader{
				Header: "X-Proxy-Auth",
				RequiredValue: corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "argo-header-auth",
					},
					Key: "proxy-secret",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "argo-header-auth",
				},
				Data: map[string][]byte{
					"proxy-secret": {},
				},
			},
			expectedError: "shared secret value is empty",
		},
		{
			name: "secret key is missing",
			cfg: &config.SharedSecretHeader{
				Header: "X-Proxy-Auth",
				RequiredValue: corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "argo-header-auth",
					},
					Key: "proxy-secret",
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "argo-header-auth",
				},
				Data: map[string][]byte{
					"other-key": []byte("secret"),
				},
			},
			expectedError: "key proxy-secret missing in secret argo-header-auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			if tt.secret != nil {
				_, err := clientset.CoreV1().
					Secrets("default").
					Create(context.Background(), tt.secret, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			got, err := getHeaderSharedSecret(
				context.Background(),
				clientset.CoreV1().Secrets("default"),
				tt.cfg,
			)

			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}
