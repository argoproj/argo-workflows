package fields

import (
	"testing"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var sampleWorkflow = `
metadata:
  name: hello-world-qgpxz
spec:
  entrypoint: whalesay
status:
  nodes:
    hello-world-qgpxz:
      displayName: hello-world-qgpxz
  phase: Succeeded
`

func TestCleaner_WillExclude(t *testing.T) {
	t.Run("Noop", func(t *testing.T) {
		require.False(t, NewCleaner("").WillExclude("foo"), "special case - keep everything")
	})
	t.Run("Default", func(t *testing.T) {
		require.False(t, NewCleaner("foo").WillExclude("foo"))
		require.False(t, NewCleaner("foo").WillExclude("foo.bar"))
		require.True(t, NewCleaner("foo").WillExclude("bar"))
		require.False(t, NewCleaner("foo.bar.baz").WillExclude("foo.bar"))

	})
	t.Run("Exclude", func(t *testing.T) {
		require.True(t, NewCleaner("-foo").WillExclude("foo"))
		require.True(t, NewCleaner("-foo").WillExclude("foo.bar"))
		require.False(t, NewCleaner("-foo").WillExclude("bar"))
		require.False(t, NewCleaner("-foo").WillExclude("bar.baz"))
	})
}

func TestCleaner_WithPrefix(t *testing.T) {
	cleaner := NewCleaner("result.object.status").WithoutPrefix("result.object.")
	require.False(t, cleaner.fields["result.object.status"])
	require.True(t, cleaner.fields["status"])
}

func TestCleanNoop(t *testing.T) {
	var wf, cleanWf wfv1.Workflow
	ok, err := NewCleaner("").Clean(wf, cleanWf)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestCleanFields(t *testing.T) {
	var wf, cleanWf wfv1.Workflow
	wfv1.MustUnmarshal([]byte(sampleWorkflow), &wf)
	ok, err := NewCleaner("status.phase,metadata.name,spec.entrypoint").Clean(wf, &cleanWf)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, wfv1.WorkflowSucceeded, cleanWf.Status.Phase)
	require.Equal(t, "whalesay", cleanWf.Spec.Entrypoint)
	require.Equal(t, "hello-world-qgpxz", cleanWf.Name)
	require.Nil(t, cleanWf.Status.Nodes)
}

func TestCleanFieldsExclude(t *testing.T) {
	var wf, cleanWf wfv1.Workflow
	wfv1.MustUnmarshal([]byte(sampleWorkflow), &wf)
	ok, err := NewCleaner("-status.phase,metadata.name,spec.entrypoint").Clean(wf, &cleanWf)
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, cleanWf.Status.Phase)
	require.Empty(t, cleanWf.Spec.Entrypoint)
	require.Empty(t, cleanWf.Name)
	require.NotNil(t, cleanWf.Status.Nodes)
}
