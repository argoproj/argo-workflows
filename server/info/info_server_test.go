package info

import (
	"context"
	"testing"

	"github.com/coreos/go-oidc"
	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo/server/auth"
)

func Test_infoServer_GetUserInfo(t *testing.T) {
	i := &infoServer{}
	ctx := context.WithValue(context.TODO(), auth.UserInfoKey, &oidc.UserInfo{Subject: "my-sub"})
	info, err := i.GetUserInfo(ctx, nil)
	if assert.NoError(t, err) {
		assert.Equal(t, "my-sub", info.Subject)
	}
}
