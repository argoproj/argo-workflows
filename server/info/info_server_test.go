package info

import (
	"context"
	"testing"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
)

func Test_infoServer_GetUserInfo(t *testing.T) {
	i := &infoServer{}
	ctx := context.WithValue(context.TODO(), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Issuer: "my-iss", Subject: "my-sub"}, Groups: []string{"my-group"}, Email: "my@email", EmailVerified: true, ServiceAccountName: "my-sa"})
	info, err := i.GetUserInfo(ctx, nil)
	if assert.NoError(t, err) {
		assert.Equal(t, "my-iss", info.Issuer)
		assert.Equal(t, "my-sub", info.Subject)
		assert.Equal(t, []string{"my-group"}, info.Groups)
		assert.Equal(t, "my@email", info.Email)
		assert.True(t, info.EmailVerified)
		assert.Equal(t, "my-sa", info.ServiceAccountName)
	}
}

func Test_infoServer_GetInfo(t *testing.T) {
	t.Run("Ful Fields", func(t *testing.T) {
		i := &infoServer{
			managedNamespace: "argo",
			links: []*wfv1.Link{
				{Name: "link-name", Scope: "scope", URL: "https://example.com"},
			},
			navColor: "red",
		}
		info, err := i.GetInfo(context.TODO(), nil)
		if assert.NoError(t, err) {
			assert.Equal(t, "argo", info.ManagedNamespace)
			assert.Equal(t, "link-name", info.Links[0].Name)
			assert.Equal(t, "red", info.NavColor)
		}
	})

	t.Run("Min Fields", func(t *testing.T) {
		i := &infoServer{}
		info, err := i.GetInfo(context.TODO(), nil)
		if assert.NoError(t, err) {
			assert.Equal(t, "", info.ManagedNamespace)
			assert.Equal(t, 0, len(info.Links))
			assert.Equal(t, "", info.NavColor)
		}
	})
}
