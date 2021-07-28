package info

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
)

func Test_infoServer_GetUserInfo(t *testing.T) {
	i := &infoServer{}
	tGroup := []string{"my-group"}
	tGroupsIf := make([]interface{}, len(tGroup))
	for i := range tGroup {
		tGroupsIf[i] = tGroup[i]
	}
	t.Run("AllClaimsSet", func(t *testing.T) {
		claims := types.Claims{
			"iss":                 "my-iss",
			"sub":                 "my-sub",
			"groups":              tGroupsIf,
			"email":               "my@email",
			"email_verified":      true,
			"serviceaccount_name": "my-sa",
		}
		ctx := context.WithValue(context.TODO(), auth.ClaimsKey, claims)
		info, err := i.GetUserInfo(ctx, nil)
		if assert.NoError(t, err) {
			assert.Equal(t, "my-iss", info.Issuer)
			assert.Equal(t, "my-sub", info.Subject)
			assert.Equal(t, []string{"my-group"}, info.Groups)
			assert.Equal(t, "my@email", info.Email)
			assert.True(t, info.EmailVerified)
			assert.Equal(t, "my-sa", info.ServiceAccountName)
		}
	})
	t.Run("EmailClaimMissing", func(t *testing.T) {
		claims := types.Claims{
			"iss":                 "my-iss",
			"sub":                 "my-sub",
			"groups":              tGroupsIf,
			"email_verified":      true,
			"serviceaccount_name": "my-sa",
		}
		ctx := context.WithValue(context.TODO(), auth.ClaimsKey, claims)
		info, err := i.GetUserInfo(ctx, nil)
		if assert.NoError(t, err) {
			assert.Equal(t, "my-iss", info.Issuer)
			assert.Equal(t, "my-sub", info.Subject)
			assert.Equal(t, []string{"my-group"}, info.Groups)
			assert.Equal(t, "", info.Email)
			assert.True(t, info.EmailVerified)
			assert.Equal(t, "my-sa", info.ServiceAccountName)
		}
	})
	t.Run("GroupsClaimMissing", func(t *testing.T) {
		claims := types.Claims{
			"iss":                 "my-iss",
			"sub":                 "my-sub",
			"email":               "my@email",
			"email_verified":      true,
			"serviceaccount_name": "my-sa",
		}
		ctx := context.WithValue(context.TODO(), auth.ClaimsKey, claims)
		info, err := i.GetUserInfo(ctx, nil)
		if assert.NoError(t, err) {
			assert.Equal(t, "my-iss", info.Issuer)
			assert.Equal(t, "my-sub", info.Subject)
			assert.Equal(t, []string{}, info.Groups)
			assert.Equal(t, "my@email", info.Email)
			assert.True(t, info.EmailVerified)
			assert.Equal(t, "my-sa", info.ServiceAccountName)
		}
	})
	t.Run("CustomGroupsClaims", func(t *testing.T) {
		tGroup := []string{"my-group"}
		tGroupsIf := make([]interface{}, len(tGroup))
		for i := range tGroup {
			tGroupsIf[i] = tGroup[i]
		}
		claims := types.Claims{
			"iss":                 "my-iss",
			"sub":                 "my-sub",
			"groupname":           tGroupsIf,
			"email":               "my@email",
			"email_verified":      true,
			"serviceaccount_name": "my-sa",
		}
		ctx := context.WithValue(context.TODO(), auth.ClaimsKey, claims)
		info, err := i.GetUserInfo(ctx, nil)
		if assert.NoError(t, err) {
			assert.Equal(t, "my-iss", info.Issuer)
			assert.Equal(t, "my-sub", info.Subject)
			assert.Equal(t, []string{"my-group"}, info.Groups)
			assert.Equal(t, "my@email", info.Email)
			assert.True(t, info.EmailVerified)
			assert.Equal(t, "my-sa", info.ServiceAccountName)
		}
	})
}
