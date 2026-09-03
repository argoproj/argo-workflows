package header

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/argoproj/argo-workflows/v4/config"
)

func TestAuthorize(t *testing.T) {
	tests := []struct {
		name string

		cfg config.HeaderConfig
		md  metadata.MD

		sharedSecret         string
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
				},
				Subject: config.ClaimSource{
					Header: "X-Forwarded-User",
				},
			},

			md: metadata.Pairs(
				"x-proxy-auth", "secret",
				"x-forwarded-user", "pradeep",
			),

			sharedSecret: "secret",

			subject: "pradeep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(tt.cfg, tt.sharedSecret, tt.trustUnauthenticated)

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
			h := New(tt.cfg, "", true)

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

	h := New(cfg, "", true)

	claims, err := h.Authorize(metadata.MD{})

	assert.Nil(t, claims)
	assert.EqualError(t, err, "subject claim is empty")
}


func TestAuthenticateProxy(t *testing.T) {
	tests := []struct {
		name          string
		cfg           config.HeaderConfig
		sharedSecret  string
		md            metadata.MD
		expectedError string
	}{
		{
			name: "valid secret",
			cfg: config.HeaderConfig{
				SharedSecret: &config.SharedSecretHeader{
					Header: "X-Proxy-Auth",
				},
			},
			sharedSecret: "secret",
			md: metadata.Pairs(
				"x-proxy-auth", "secret",
			),
		},
		{
			name: "invalid secret",
			cfg: config.HeaderConfig{
				SharedSecret: &config.SharedSecretHeader{
					Header: "X-Proxy-Auth",
				},
			},
			sharedSecret: "secret",
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
				},
			},
			sharedSecret:  "secret",
			md:            metadata.MD{},
			expectedError: "trusted proxy authentication header is missing",
		},
		{
			name:          "shared secret not configured",
			cfg:           config.HeaderConfig{},
			sharedSecret:  "secret",
			md:            metadata.Pairs("x-proxy-auth", "secret"),
			expectedError: "shared secret authentication is not configured",
		},
		{
			name: "one of multiple authentication headers matches",
			cfg: config.HeaderConfig{
				SharedSecret: &config.SharedSecretHeader{
					Header: "X-Proxy-Auth",
				},
			},
			sharedSecret: "secret",
			md: metadata.Pairs(
				"x-proxy-auth", "wrong-secret",
				"x-proxy-auth", "secret",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(tt.cfg, tt.sharedSecret, false)

			err := h.(*header).authenticateProxy(tt.md)

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
		},
		Subject: config.ClaimSource{
			Header: "X-Forwarded-User",
		},
	}

	t.Run("secure mode requires valid proxy secret", func(t *testing.T) {
		h := New(cfg, "secret", false)

		claims, err := h.Authorize(metadata.Pairs(
			"x-proxy-auth", "secret",
			"x-forwarded-user", "pradeep",
		))

		require.NoError(t, err)
		assert.Equal(t, "pradeep", claims.Subject)
	})

	t.Run("secure mode rejects invalid proxy secret", func(t *testing.T) {
		h := New(cfg, "secret", false)

		claims, err := h.Authorize(metadata.Pairs(
			"x-proxy-auth", "wrong-secret",
			"x-forwarded-user", "pradeep",
		))

		assert.Nil(t, claims)
		require.EqualError(t, err, "trusted proxy authentication failed")
	})

	t.Run("secure mode rejects missing proxy secret", func(t *testing.T) {
		h := New(cfg, "secret", false)

		claims, err := h.Authorize(metadata.Pairs(
			"x-forwarded-user", "pradeep",
		))

		assert.Nil(t, claims)
		require.EqualError(t, err, "trusted proxy authentication header is missing")
	})

	t.Run("insecure mode skips proxy authentication", func(t *testing.T) {
		h := New(cfg, "", true)

		claims, err := h.Authorize(metadata.Pairs(
			"x-forwarded-user", "pradeep",
		))

		require.NoError(t, err)
		assert.Equal(t, "pradeep", claims.Subject)
	})
}