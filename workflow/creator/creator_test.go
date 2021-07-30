package creator

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2/jwt"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func TestLabel(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		Label(context.TODO(), wf)
		assert.Empty(t, wf.Labels)
	})
	t.Run("NotEmpty", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		Label(context.WithValue(context.TODO(), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: strings.Repeat("x", 63) + "y"}, Email: "my@email"}), wf)
		if assert.NotEmpty(t, wf.Labels) {
			assert.Equal(t, strings.Repeat("x", 62)+"y", wf.Labels[common.LabelKeyCreator], "creator is truncated")
			assert.Equal(t, "my.at.email", wf.Labels[common.LabelKeyCreatorEmail], "'@' is replaced by '.at.'")
		}
	})
	t.Run("TooLongHyphen", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		Label(context.WithValue(context.TODO(), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: strings.Repeat("-", 63) + "y"}}), wf)
		if assert.NotEmpty(t, wf.Labels) {
			assert.Equal(t, "y", wf.Labels[common.LabelKeyCreator])
		}
	})
	t.Run("InvalidDNSNames", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		Label(context.WithValue(context.TODO(), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: "!@#$%^&*()--__" + strings.Repeat("y", 35) + "__--!@#$%^&*()"}}), wf)
		if assert.NotEmpty(t, wf.Labels) {
			assert.Equal(t, strings.Repeat("y", 35), wf.Labels[common.LabelKeyCreator])
		}
	})
	t.Run("InvalidDNSNamesWithMidDashes", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		sub := strings.Repeat("x", 20) + strings.Repeat("-", 70) + strings.Repeat("x", 20)
		Label(context.WithValue(context.TODO(), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: sub}}), wf)
		if assert.NotEmpty(t, wf.Labels) {
			assert.Equal(t, strings.Repeat("x", 20), wf.Labels[common.LabelKeyCreator])
		}
	})
}
