package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestMergeWorkflows(t *testing.T) {
	patchWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/orig-wf.yaml")
	targetWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch.yaml")

	err := MergeTo(patchWf, targetWf)
	require.NoError(t, err)
	assert.Equal(t, "start", targetWf.Spec.Entrypoint)
	assert.Equal(t, "argo1", targetWf.Spec.ServiceAccountName)
	assert.Equal(t, "message", targetWf.Spec.Arguments.Parameters[0].Name)
	assert.Equal(t, "patch", targetWf.Spec.Arguments.Parameters[0].Value.String())
}

func TestMergeHooks(t *testing.T) {
	t.Run("NilBaseAndNilPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch-nil-hook.yaml")
		targetHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/base-nil.yaml")

		err := MergeTo(patchHookWf, targetHookWf)
		require.NoError(t, err)
		assert.Nil(t, targetHookWf.Spec.Hooks)
	})

	t.Run("NilBaseAndNotNilPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch-hook.yaml")
		targetHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/base-nil.yaml")

		err := MergeTo(patchHookWf, targetHookWf)
		require.NoError(t, err)
		assert.Len(t, targetHookWf.Spec.Hooks, 2)
		assert.Equal(t, "c", targetHookWf.Spec.Hooks[`foo`].Template)
		assert.Equal(t, "b", targetHookWf.Spec.Hooks[`bar`].Template)
	})

	// Ensure hook bar ends up in result, but foo is unchanged
	t.Run("NotNilBaseAndPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch-hook.yaml")
		targetHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/base-hook.yaml")

		err := MergeTo(patchHookWf, targetHookWf)
		require.NoError(t, err)
		assert.Len(t, targetHookWf.Spec.Hooks, 2)
		assert.Equal(t, "a", targetHookWf.Spec.Hooks[`foo`].Template)
		assert.Equal(t, "b", targetHookWf.Spec.Hooks[`bar`].Template)
	})
}

func TestMergeLabelsFrom(t *testing.T) {
	t.Run("NilBaseAndNotNilPatch", func(t *testing.T) {
		patchWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch-labels-from.yaml")
		targetWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/base-nil.yaml")

		err := MergeTo(patchWf, targetWf)
		require.NoError(t, err)
		assert.Len(t, targetWf.Spec.WorkflowMetadata.LabelsFrom, 2)
		assert.Equal(t, "PATCH", targetWf.Spec.WorkflowMetadata.LabelsFrom[`foo`].Expression)
		assert.Equal(t, "PATCH", targetWf.Spec.WorkflowMetadata.LabelsFrom[`bar`].Expression)
	})

	t.Run("NotNilBaseAndPatch", func(t *testing.T) {
		patchWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch-labels-from.yaml")
		targetWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/base-labels-from.yaml")

		err := MergeTo(patchWf, targetWf)
		require.NoError(t, err)
		assert.Len(t, targetWf.Spec.WorkflowMetadata.LabelsFrom, 3)
		assert.Equal(t, "BASE", targetWf.Spec.WorkflowMetadata.LabelsFrom[`foo`].Expression)
		assert.Equal(t, "PATCH", targetWf.Spec.WorkflowMetadata.LabelsFrom[`bar`].Expression)
		assert.Equal(t, "BASE", targetWf.Spec.WorkflowMetadata.LabelsFrom[`baz`].Expression)
	})
}

func TestMergeWorkflowSpecs(t *testing.T) {
	got, err := MergeWorkflowSpecs(
		&wfv1.WorkflowSpec{Entrypoint: "wf", ServiceAccountName: "wf-sa"},
		nil,
		&wfv1.WorkflowSpec{Entrypoint: "template", Parallelism: new(int64(2))},
		&wfv1.WorkflowSpec{Entrypoint: "defaults", ServiceAccountName: "default-sa", PodPriorityClassName: "low"},
	)
	require.NoError(t, err)
	assert.Equal(t, &wfv1.WorkflowSpec{Entrypoint: "wf", ServiceAccountName: "wf-sa", Parallelism: new(int64(2)), PodPriorityClassName: "low"}, got,
		"each spec fills in only what the ones before it leave unset, and a nil spec is skipped")

	got, err = MergeWorkflowSpecs(nil, &wfv1.WorkflowSpec{Entrypoint: "template"})
	require.NoError(t, err)
	assert.Nil(t, got, "nothing to merge into")
}
