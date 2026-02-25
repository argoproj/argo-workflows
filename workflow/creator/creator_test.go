package creator

import (
	"context"
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	"github.com/argoproj/argo-workflows/v4/server/auth/types"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestLabelCreator(t *testing.T) {
	t.Run("EmptyCreator", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		LabelCreator(logging.TestContext(t.Context()), wf)
		assert.Empty(t, wf.Labels)
	})
	t.Run("EmptyActor", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		LabelActor(logging.TestContext(t.Context()), wf, ActionResume)
		assert.Empty(t, wf.Labels)
	})
	t.Run("NotEmptyCreator", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		LabelCreator(context.WithValue(logging.TestContext(t.Context()), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: strings.Repeat("x", 63) + "y"}, Email: "my@email", PreferredUsername: "username"}), wf)
		require.NotEmpty(t, wf.Labels)
		assert.Equal(t, strings.Repeat("x", 62)+"y", wf.Labels[common.LabelKeyCreator], "creator is truncated")
		assert.Equal(t, "my.at.email", wf.Labels[common.LabelKeyCreatorEmail], "'@' is replaced by '.at.'")
		assert.Equal(t, "username", wf.Labels[common.LabelKeyCreatorPreferredUsername], "username is matching")
		assert.Empty(t, wf.Labels[common.LabelKeyAction])
	})
	t.Run("NotEmptyActor", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		LabelActor(context.WithValue(logging.TestContext(t.Context()), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: strings.Repeat("x", 63) + "y"}, Email: "my@email", PreferredUsername: "username"}), wf, ActionResume)
		require.NotEmpty(t, wf.Labels)
		assert.Equal(t, strings.Repeat("x", 62)+"y", wf.Labels[common.LabelKeyActor], "creator is truncated")
		assert.Equal(t, "my.at.email", wf.Labels[common.LabelKeyActorEmail], "'@' is replaced by '.at.'")
		assert.Equal(t, "username", wf.Labels[common.LabelKeyActorPreferredUsername], "username is matching")
		assert.Equal(t, "Resume", wf.Labels[common.LabelKeyAction])
	})
	t.Run("TooLongHyphen", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		LabelCreator(context.WithValue(logging.TestContext(t.Context()), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: strings.Repeat("-", 63) + "y"}}), wf)
		require.NotEmpty(t, wf.Labels)
		assert.Equal(t, "y", wf.Labels[common.LabelKeyCreator])
	})
	t.Run("InvalidDNSNames", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		LabelCreator(context.WithValue(logging.TestContext(t.Context()), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: "!@#$%^&*()--__" + strings.Repeat("y", 35) + "__--!@#$%^&*()"}, PreferredUsername: "us#er@name#"}), wf)
		require.NotEmpty(t, wf.Labels)
		assert.Equal(t, strings.Repeat("y", 35), wf.Labels[common.LabelKeyCreator])
		assert.Equal(t, "us-er-name", wf.Labels[common.LabelKeyCreatorPreferredUsername], "username is truncated")
	})
	t.Run("InvalidDNSNamesWithMidDashes", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		sub := strings.Repeat("x", 20) + strings.Repeat("-", 70) + strings.Repeat("x", 20)
		LabelCreator(context.WithValue(logging.TestContext(t.Context()), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: sub}}), wf)
		require.NotEmpty(t, wf.Labels)
		assert.Equal(t, strings.Repeat("x", 20), wf.Labels[common.LabelKeyCreator])
	})
	t.Run("DifferentUsersFromCreatorLabels", func(t *testing.T) {
		type input struct {
			claims *types.Claims
			wf     *wfv1.Workflow
		}
		type output struct {
			creatorLabelsToHave    map[string]string
			creatorLabelsNotToHave []string
		}
		for _, testCase := range []struct {
			name   string
			input  *input
			output *output
		}{
			{
				name: "when claims are empty",
				input: &input{
					claims: &types.Claims{Claims: jwt.Claims{}},
					wf: &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
						common.LabelKeyCreator:                  "xxxx-xxxx-xxxx-xxxx",
						common.LabelKeyCreatorEmail:             "foo.at.example.com",
						common.LabelKeyCreatorPreferredUsername: "foo",
					}}},
				},
				output: &output{
					creatorLabelsToHave:    nil,
					creatorLabelsNotToHave: []string{common.LabelKeyCreator, common.LabelKeyCreatorEmail, common.LabelKeyCreatorPreferredUsername},
				},
			}, {
				name: "when claims are nil",
				input: &input{
					claims: nil,
					wf: &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
						common.LabelKeyCreator:                  "xxxx-xxxx-xxxx-xxxx",
						common.LabelKeyCreatorEmail:             "foo.at.example.com",
						common.LabelKeyCreatorPreferredUsername: "foo",
					}}},
				},
				output: &output{
					creatorLabelsToHave:    nil,
					creatorLabelsNotToHave: []string{common.LabelKeyCreator, common.LabelKeyCreatorEmail, common.LabelKeyCreatorPreferredUsername},
				},
			}, {
				name: "when user information in claim is different from the existing labels of a Workflow",
				input: &input{
					claims: &types.Claims{Claims: jwt.Claims{Subject: "yyyy-yyyy-yyyy-yyyy"}, Email: "bar.at.example.com", PreferredUsername: "bar"},
					wf: &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
						common.LabelKeyCreator:                  "xxxx-xxxx-xxxx-xxxx",
						common.LabelKeyCreatorEmail:             "foo.at.example.com",
						common.LabelKeyCreatorPreferredUsername: "foo",
					}}},
				},
				output: &output{
					creatorLabelsToHave: map[string]string{
						common.LabelKeyCreator:                  "yyyy-yyyy-yyyy-yyyy",
						common.LabelKeyCreatorEmail:             "bar.at.example.com",
						common.LabelKeyCreatorPreferredUsername: "bar",
					},
					creatorLabelsNotToHave: nil,
				},
			},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				LabelCreator(context.WithValue(logging.TestContext(t.Context()), auth.ClaimsKey, testCase.input.claims), testCase.input.wf)
				labels := testCase.input.wf.GetLabels()
				for k, expectedValue := range testCase.output.creatorLabelsToHave {
					assert.Equal(t, expectedValue, labels[k])
				}
				for _, creatorLabelKey := range testCase.output.creatorLabelsNotToHave {
					_, ok := labels[creatorLabelKey]
					assert.Falsef(t, ok, "should not have the creator label, \"%s\"", creatorLabelKey)
				}
			})

		}
	})
}

func TestUserInfoMap(t *testing.T) {
	t.Run("NotEmpty", func(t *testing.T) {
		ctx := context.WithValue(logging.TestContext(t.Context()), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: strings.Repeat("x", 63) + "y"}, Email: "my@email", PreferredUsername: "username"})
		uim := UserInfoMap(ctx)
		assert.Equal(t, map[string]string{
			"User":              strings.Repeat("x", 63) + "y",
			"Email":             "my@email",
			"PreferredUsername": "username",
		}, uim)
	})
	t.Run("Empty", func(t *testing.T) {
		uim := UserInfoMap(logging.TestContext(t.Context()))
		assert.Nil(t, uim)
	})
}
