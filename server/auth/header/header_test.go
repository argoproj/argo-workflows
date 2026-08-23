package header

import (
	"testing"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestAuthorize(t *testing.T) {
	tests := []struct {
		name string

		cfg config.HeaderConfig
		md  metadata.MD

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

			subject: "pradeep",
			email:   "abc@test.com",
		},

		{
			name: "groups",

			cfg: config.HeaderConfig{
				Groups: config.GroupClaimSource{
					ClaimSource: config.ClaimSource{
						Header: "X-Forwarded-Groups",
					},
					Delimiter: ",",
				},
			},

			md: metadata.Pairs(
				"x-forwarded-groups",
				"admin,developer,argo",
			),

			groups: []string{
				"admin",
				"developer",
				"argo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := New(tt.cfg)

			claims, err := h.Authorize(tt.md)

			assert.NoError(t, err)
			assert.Equal(t, tt.issuer, claims.Claims.Issuer)
			assert.Equal(t, tt.subject, claims.Claims.Subject)
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
			h := New(tt.cfg)

			assert.Equal(t, tt.want, h.IsRBACEnabled())
		})
	}
}
