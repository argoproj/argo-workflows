package info

import (
	"context"
	"testing"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
)

func Test_infoServer_GetUserInfo(t *testing.T) {
	i := &infoServer{}
	ctx := context.WithValue(context.TODO(), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Issuer: "my-iss", Subject: "my-sub"}, Groups: []string{"my-group"}, Name: "myname", Email: "my@email", EmailVerified: true, ServiceAccountName: "my-sa"})
	info, err := i.GetUserInfo(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, "my-iss", info.Issuer)
	require.Equal(t, "my-sub", info.Subject)
	require.Equal(t, []string{"my-group"}, info.Groups)
	require.Equal(t, "myname", info.Name)
	require.Equal(t, "my@email", info.Email)
	require.True(t, info.EmailVerified)
	require.Equal(t, "my-sa", info.ServiceAccountName)
}

func Test_infoServer_GetInfo(t *testing.T) {
	t.Run("Ful Fields", func(t *testing.T) {
		i := &infoServer{
			managedNamespace: "argo",
			links: []*wfv1.Link{
				{Name: "link-name", Scope: "scope", URL: "https://example.com"},
			},
			columns: []*wfv1.Column{
				{Name: "Workflow Completed", Type: "label", Key: "workflows.argoproj.io/completed"},
			},
			navColor: "red",
		}
		info, err := i.GetInfo(context.TODO(), nil)
		require.NoError(t, err)
		require.Equal(t, "argo", info.ManagedNamespace)
		require.Equal(t, "link-name", info.Links[0].Name)
		require.Equal(t, "red", info.NavColor)
		require.Equal(t, "Workflow Completed", info.Columns[0].Name)
		require.Equal(t, "label", info.Columns[0].Type)
		require.Equal(t, "workflows.argoproj.io/completed", info.Columns[0].Key)
	})

	t.Run("Min Fields", func(t *testing.T) {
		i := &infoServer{}
		info, err := i.GetInfo(context.TODO(), nil)
		require.NoError(t, err)
		require.Equal(t, "", info.ManagedNamespace)
		require.Empty(t, info.Links)
		require.Empty(t, info.Columns)
		require.Equal(t, "", info.NavColor)
	})
}
