package v1alpha1

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// TestLifecycleHookExpressionDescription guards against a regression where the
// OpenAPI description of `LifecycleHook.expression` was copy-pasted from
// `RetryStrategy.expression` (see argoproj/argo-workflows#15007). The two
// fields have unrelated semantics: LifecycleHook.expression gates whether the
// hook fires; RetryStrategy.expression gates whether a retry occurs. The
// description must not talk about retries.
func TestLifecycleHookExpressionDescription(t *testing.T) {
	defs := GetOpenAPIDefinitions(func(path string) spec.Ref { return spec.Ref{} })

	hook, ok := defs["github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1.LifecycleHook"]
	if !assert.True(t, ok, "LifecycleHook missing from OpenAPI definitions") {
		return
	}
	exprProp, ok := hook.Schema.Properties["expression"]
	if !assert.True(t, ok, "LifecycleHook.expression missing from schema") {
		return
	}
	desc := exprProp.Description

	// The bug: the description used to contain "retried" / "retry strategy".
	// Both phrases belong to RetryStrategy, not LifecycleHook.
	assert.NotContains(t, strings.ToLower(desc), "retried",
		"LifecycleHook.expression description must not mention retries (was copy-pasted from RetryStrategy)")
	assert.NotContains(t, strings.ToLower(desc), "retry strategy",
		"LifecycleHook.expression description must not mention retry strategy")
	// And it should actually describe the hook semantics.
	assert.Contains(t, strings.ToLower(desc), "hook",
		"LifecycleHook.expression description should mention hook semantics")

	// Sanity check: RetryStrategy.expression *must* still mention retries —
	// guards us from a future patch over-correcting and damaging the right
	// description.
	retry, ok := defs["github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1.RetryStrategy"]
	if !assert.True(t, ok, "RetryStrategy missing from OpenAPI definitions") {
		return
	}
	retryExpr, ok := retry.Schema.Properties["expression"]
	if !assert.True(t, ok, "RetryStrategy.expression missing from schema") {
		return
	}
	assert.Contains(t, strings.ToLower(retryExpr.Description), "retried",
		"RetryStrategy.expression description should still describe retries")
}

// silence "unused" warnings on the common import if the test file is split
// across builds.
var _ = common.ReferenceCallback(nil)
